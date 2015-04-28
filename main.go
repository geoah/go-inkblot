package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/RangelReale/osin"
	"github.com/RangelReale/osin/example"
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
var db *mgo.Database

// var initIdentity bool = false

var self Identity

// var initURIsString string = ""
var localPort uint = 0
var server *osin.Server

// var identityURL string = ""
// var initURIs []string = make([]string, 0)

// func init() {
// flag.StringVar(&identityURL, "id", "", "Identity URL")
// flag.StringVar(&self.Hostname, "hostname", "localhost", "Hostname")
// flag.UintVar(&self.Port, "port", 9000, "Port")
// flag.BoolVar(&self.UseSSL, "ssl", false, "SSL")
// flag.BoolVar(&initIdentity, "init", false, "Create Identity")
// flag.StringVar(&initURIsString, "ids", "", "Initial Identities to connect to")
// }

type MyServer struct {
	r *mux.Router
}

func (s *MyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println(">>>>> HTTP")
	// if origin := req.Header.Get("Origin"); origin != "" {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Set("Access-Control-Allow-Headers",
		"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	// }
	// Stop here if its Preflighted OPTIONS request
	if req.Method == "OPTIONS" {
		return
	}
	// Lets Gorilla work
	s.r.ServeHTTP(rw, req)
}

func main() {
	// Parse flags
	flag.Parse()

	// if os.Getenv("INK_IDENTITY_URL") != "" {
	// 	identityURL = os.Getenv("INK_IDENTITY_URL")
	// }

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/init", HandlePublicInit).Methods("GET")
	router.HandleFunc("/", HandlePublicIndex).Methods("GET")
	router.HandleFunc("/", HandlePublicIndexPost).Methods("POST")
	router.Handle("/instances", HandleOwnOrIdentity(http.HandlerFunc(HandleOwnInstancesPost), http.HandlerFunc(HandleIdentityInstancesPost))).Methods("POST")
	router.Handle("/instances", http.HandlerFunc(HandleOwnInstances)).Methods("GET")
	router.Handle("/identities", HandleOwn(http.HandlerFunc(HandleOwnIdentities))).Methods("GET")
	router.Handle("/identities", HandleOwn(http.HandlerFunc(HandleOwnIdentitiesPost))).Methods("POST")
	router.Handle("/settings", HandleOwn(http.HandlerFunc(HandleOwnSettings))).Methods("GET")

	sconfig := osin.NewServerConfig()
	sconfig.AllowedAuthorizeTypes = osin.AllowedAuthorizeType{osin.CODE, osin.TOKEN}
	sconfig.AllowedAccessTypes = osin.AllowedAccessType{osin.AUTHORIZATION_CODE, osin.REFRESH_TOKEN, osin.PASSWORD, osin.CLIENT_CREDENTIALS, osin.ASSERTION}
	sconfig.AllowGetAccessRequest = true
	sconfig.ErrorStatusCode = 401

	server = osin.NewServer(sconfig, NewTestStorage())

	// Authorization code endpoint
	router.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		resp := server.NewResponse()
		defer resp.Close()

		if ar := server.HandleAuthorizeRequest(resp, r); ar != nil {
			if !example.HandleLoginPage(ar, w, r) {
				return
			}
			ar.UserData = struct{ Login string }{Login: "test"}
			ar.Authorized = true
			server.FinishAuthorizeRequest(resp, r, ar)
		}
		if resp.IsError && resp.InternalError != nil {
			fmt.Printf("ERROR: %s\n", resp.InternalError)
		}
		if !resp.IsError {
			resp.Output["custom_parameter"] = 187723
		}
		osin.OutputJSON(resp, w, r)
	})

	// Access token endpoint
	router.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		resp := server.NewResponse()
		defer resp.Close()

		if ar := server.HandleAccessRequest(resp, r); ar != nil {
			switch ar.Type {
			case osin.AUTHORIZATION_CODE:
				ar.Authorized = true
			case osin.REFRESH_TOKEN:
				ar.Authorized = true
			case osin.PASSWORD:
				if ar.Username == "test" && ar.Password == "test" {
					ar.Authorized = true
				}
			case osin.CLIENT_CREDENTIALS:
				ar.Authorized = true
			case osin.ASSERTION:
				if ar.AssertionType == "urn:osin.example.complete" && ar.Assertion == "osin.data" {
					ar.Authorized = true
				}
			}
			server.FinishAccessRequest(resp, r, ar)
		}
		if resp.IsError && resp.InternalError != nil {
			fmt.Printf("ERROR: %s\n", resp.InternalError)
		}
		if !resp.IsError {
			resp.Output["custom_parameter"] = 19923
		}
		osin.OutputJSON(resp, w, r)
	})

	// Information endpoint
	router.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		resp := server.NewResponse()
		defer resp.Close()
		ir := server.HandleInfoRequest(resp, r)
		if ir != nil {
			server.FinishInfoRequest(resp, r, ir)
		}
		osin.OutputJSON(resp, w, r)
	})

	// r := mux.NewRouter()
	http.Handle("/", &MyServer{router})
	// http.ListenAndServe(":14000", nil)

	go func() {
		// Check that the id url has been set
		// if os.Getenv("INK_HOSTNAME") == "" {
		// 	log.Fatal("Missing INK_HOSTNAME")
		// }

		// Fetch self id
		// self, err := FetchSelfIdentity(identityURL)
		// if err != nil {
		// 	panic(err)
		// }

		if os.Getenv("MONGOLAB_URI") != "" {
			session, err := mgo.Dial(os.Getenv("MONGOLAB_URI"))
			if err != nil {
				panic(err)
			}
			db = session.DB("") //.C("people")
			// err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
			// 	&Person{"Cla", "+55 53 8402 8510"})
			// if err != nil {
			// 	log.Fatal(err)
			// }
			//

			hostnameKv := KV{}
			err = db.C("settings").Find(bson.M{"_id": "hostname"}).One(&hostnameKv)
			if err != nil {
				fmt.Println("You need to init this instance")
			} else {
				self = Identity{}
				err := db.C("identities").Find(bson.M{"hostname": hostnameKv.Value}).One(&self)
				if err == nil {
					fmt.Println("Could not find identity")
				}
				// rt.self = &self
			}
		} else {
			log.Fatal(errors.New("Missing db connection"))
		}
		//defer session.Close()

		// Show own URI
		fmt.Printf("Starting up on %d\n", localPort)

		// if initURIsString != "" {
		// 	initURIs = strings.Split(initURIsString, ",")
		// }

		rt = newRoutingTable(&self)
		// if len(initURIs) > 0 {
		// 	for _, uri := range initURIs {
		// 		var identity Identity
		// 		identity, err := FetchIdentity(uri)
		// 		if err != nil {
		// 			fmt.Printf("Could not fetch %s, error: %s\n", uri, err)
		// 		} else {
		// 			rt.insertIdentity(&identity)
		// 		}
		// 	}
		// }
	}()

	if os.Getenv("PORT") != "" {
		tempLocalPort, _ := strconv.Atoi(os.Getenv("PORT"))
		localPort = uint(tempLocalPort)
		fmt.Println("Fast forwarding HTTP server")
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", localPort), router))
	}

	// if localPort == 0 {
	// 	localPort = self.Port
	// 	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", localPort), router))
	// }
	// fmt.Println("Ready...")

}
