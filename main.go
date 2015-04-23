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
	router.HandleFunc("/instances", HandleIdentityInstancesPost).Methods("POST")
	router.HandleFunc("/identities", HandleOwnIdentities).Methods("GET")
	router.HandleFunc("/identities", HandleOwnIdentitiesPost).Methods("POST")
	router.HandleFunc("/settings", HandleOwnSettings).Methods("GET")

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
