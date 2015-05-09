package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/RangelReale/osin"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ashkang/osin-mongo-storage/mgostore"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

type KV struct {
	Key   string `json:"key" bson:"_id"`
	Value string `json:"value" bson:"value"`
}

type routingTable struct {
	self       *Identity
	identities map[string]*Identity
	lock       *sync.RWMutex
}

func newRoutingTable(selfIdentity *Identity) *routingTable {
	return &routingTable{
		self:       selfIdentity,
		identities: map[string]*Identity{},
		lock:       new(sync.RWMutex),
	}
}

func (s *routingTable) insertIdentity(identity *Identity) error {
	// rt.identities = append(rt.identities, identity)
	s.identities[identity.ID] = identity
	return nil
}

func (s *routingTable) Get(ID string) (*Identity, error) {
	if identity, ok := s.identities[ID]; ok {
		return identity, nil
	}
	return nil, errors.New("Does not exist")
}

var rt *routingTable
var mgoSession *mgo.Session
var db *mgo.Database

// var initIdentity bool = false

var self Identity

// var initURIsString string = ""
var localPort uint = 0
var server *osin.Server

func addDefaultHeaders(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		fmt.Println(">>>>> HTTP")
		// if origin := req.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		// }
		// Stop here if its Preflighted OPTIONS request
		if r.Method != "OPTIONS" {
			fn.ServeHTTP(w, r)
		}

	}
}

func main() {
	// Parse flags
	flag.Parse()

	var err error
	mgoSession, err = mgo.Dial(getenvOrDefault("MONGOLAB_URI", "localhost"))
	if mgoSession == nil || err != nil {
		panic(err)
	}

	mainRouter := mux.NewRouter().StrictSlash(true)

	oAuth := setupOAuth(mainRouter)
	setupRestAPI(mainRouter, oAuth)

	go func() {
		if os.Getenv("MONGOLAB_URI") != "" {
			db = mgoSession.DB("") //.C("people")
			hostnameKv := KV{}
			err = db.C("settings").Find(bson.M{"_id": "hostname"}).One(&hostnameKv)
			if err != nil {
				fmt.Println("You need to init this instance")
			} else {
				self = Identity{}
				err := db.C("identities").Find(bson.M{"hostname": hostnameKv.Value}).One(&self)
				if err != nil {
					log.Fatal("Could not find identity")
				}
			}
		} else {
			log.Fatal(errors.New("Missing db connection"))
		}
	}()

	port := fmt.Sprintf(":%v", getenvOrDefault("PORT", "3000"))
	fmt.Printf("Listening on port %v\n", port)
	http.ListenAndServe(port, mainRouter)
}

func setupOAuth(router *mux.Router) *oAuthHandler {
	fmt.Println("000", mgoSession)
	oAuth := NewOAuthHandler(mgoSession, getenvOrDefault("MONGOLAB_DBNAME", ""))
	if _, err := oAuth.Storage.GetClient("1234"); err != nil {
		if _, err := setClient1234(oAuth.Storage); err != nil {
			panic(err)
		}
	}

	// oauthSub := router.PathPrefix("/").Subrouter()
	router.HandleFunc("/authorize", oAuth.AuthorizeClient)
	router.HandleFunc("/token", oAuth.GenerateToken)
	router.HandleFunc("/info", addDefaultHeaders(oAuth.HandleInfo))

	return oAuth
}

func setupRestAPI(router *mux.Router, oAuth *oAuthHandler) {
	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
		PreRoutingMiddlewares:    []rest.Middleware{oAuth},
	}
	handler.SetRoutes(
		&rest.Route{"GET", "/api/me", func(w rest.ResponseWriter, req *rest.Request) {
			data := context.Get(req.Request, USERDATA)
			w.WriteJson(&data)
		}},
	)

	router.HandleFunc("/init", HandlePublicInit).Methods("GET")
	router.HandleFunc("/", HandlePublicIndex).Methods("GET")
	router.HandleFunc("/", HandlePublicIndexPost).Methods("POST")
	router.HandleFunc("/instances", HandleOwnInstancesPost).Methods("POST")
	router.HandleFunc("/instances", HandleIdentityInstancesPost).Methods("SYNC")
	router.HandleFunc("/instances", HandleOwnInstances).Methods("GET")
	router.HandleFunc("/identities", HandleOwnIdentities).Methods("GET")
	router.HandleFunc("/identities", HandleOwnIdentitiesPost).Methods("POST")
	router.HandleFunc("/settings", HandleOwnSettings).Methods("GET")
}

func setClient1234(storage *mgostore.MongoStorage) (osin.Client, error) {
	client := &osin.DefaultClient{
		Id:          "1234",
		Secret:      "aabbccdd",
		RedirectUri: "http://localhost:9000"}
	err := storage.SetClient("1234", client)
	return client, err
}

func getenvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}
