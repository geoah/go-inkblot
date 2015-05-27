package main

import (
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/StephanDollberg/go-json-rest-middleware-jwt"
	"github.com/ant0ine/go-json-rest/rest"
)

type KV struct {
	Key   string `json:"key" bson:"_id"`
	Value string `json:"value" bson:"value"`
}

var rt *routingTable = newRoutingTable()
var mgoSession *mgo.Session
var db *mgo.Database

func main() {
	// Parse flags
	flag.Parse()

	var err error
	mgoSession, err = mgo.Dial(getenvOrDefault("MONGOLAB_URI", "localhost"))
	if mgoSession == nil || err != nil {
		panic(err)
	}

	api := rest.NewApi()
	statusMw := &rest.StatusMiddleware{}
	api.Use(statusMw)
	api.Use(rest.DefaultDevStack...)
	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,
		OriginValidator: func(origin string, request *rest.Request) bool {
			return true //origin == "http://my.other.host"
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders: []string{
			"Accept", "Authorization", "Content-Type", "X-Custom-Header", "Origin"},
		AccessControlAllowCredentials: true,
		AccessControlMaxAge:           3600,
	})

	jwtMiddleware := &jwt.JWTMiddleware{
		Key:        []byte("foobar"),
		Realm:      "jwt auth",
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(userId string, password string) bool {
			h := sha1.New()
			h.Write([]byte(password))
			passhash := h.Sum(nil)
			return rt.self.Passhash == fmt.Sprintf("%x", passhash)
		},
		SigningAlgorithm: "HS256",
	}

	jwtMiddlewareOptionally := &rest.IfMiddleware{
		Condition: func(request *rest.Request) bool {
			return request.Header.Get("Authorization") != ""
		},
		IfTrue: jwtMiddleware,
	}

	router, err := rest.MakeRouter(
		rest.Post("/login", jwtMiddleware.LoginHandler),
		rest.Get("/refresh_token", jwtMiddleware.RefreshHandler),
		rest.Post("/register", HandlePublicRegisterPost),
		rest.Get("/", jwtMiddlewareOptionally.MiddlewareFunc(HandlePublicIndex)),
		rest.Post("/", HandlePublicIndexPost),
		rest.Post("/instances", HandleOwnInstancesPost),
		// rest.HandleFunc("/instances", HandleIdentityInstancesPost).Methods("SYNC")
		rest.Get("/instances", HandleOwnInstances),
		rest.Get("/identities", jwtMiddlewareOptionally.MiddlewareFunc(HandleOwnIdentities)),
		rest.Post("/identities", HandleOwnIdentitiesPost),
		rest.Get("/settings", HandleOwnSettings),
	)
	if err != nil {
		log.Fatal(err)
	}

	api.SetApp(router)

	http.Handle("/", api.MakeHandler())
	http.Handle("/setup", http.StripPrefix("/setup", http.FileServer(http.Dir("./static/setup"))))
	http.Handle("/auth", http.StripPrefix("/auth", http.FileServer(http.Dir("./static/auth"))))
	http.Handle("/apps/friends", http.StripPrefix("/apps/friends", http.FileServer(http.Dir("./static/apps/friends"))))

	go func() {
		if os.Getenv("MONGOLAB_URI") != "" {
			db = mgoSession.DB("") //.C("people")
			hostnameKv := KV{}
			err = db.C("settings").Find(bson.M{"_id": "hostname"}).One(&hostnameKv)
			if err != nil {
				fmt.Println("You need to init this instance")
			} else {
				fmt.Println("Found setting for", hostnameKv.Value)
				self := Identity{}
				err := db.C("identities").Find(bson.M{"hostname": hostnameKv.Value}).One(&self)
				if err != nil {
					log.Fatal("Could not find identity")
				} else {
					rt.self = &self
				}
			}
		} else {
			log.Fatal(errors.New("Missing db connection"))
		}
	}()

	port := fmt.Sprintf(":%v", getenvOrDefault("PORT", "3000"))
	log.Fatal(http.ListenAndServe(port, nil))
}

func getenvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}
