package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

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
var initIdentity bool = false

var self Identity
var initURIsString string = ""
var localPort uint = 0
var identityURL string = ""
var initURIs []string = make([]string, 0)

func init() {
	flag.StringVar(&identityURL, "id", "", "Identity URL")
	// flag.StringVar(&self.Hostname, "hostname", "localhost", "Hostname")
	// flag.UintVar(&self.Port, "port", 9000, "Port")
	// flag.BoolVar(&self.UseSSL, "ssl", false, "SSL")
	flag.BoolVar(&initIdentity, "init", false, "Create Identity")
	// flag.StringVar(&initURIsString, "ids", "", "Initial Identities to connect to")
}

func main() {
	// Parse flags
	flag.Parse()

	// Check if we need to init the Identity
	if initIdentity == true {

		// if self.Hostname == "" {
		// 	log.Fatal("Missing hostname")
		// }

		var identity Identity = Identity{}
		identity.Init()
		selfJSON, err := json.Marshal(&identity)
		if err == nil {
			fmt.Println(string(selfJSON))
		}
		return
	}

	if os.Getenv("INK_IDENTITY_URL") != "" {
		identityURL = os.Getenv("INK_IDENTITY_URL")
	}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", HandlePublicIndex).Methods("GET")
	router.HandleFunc("/", HandlePublicIndexPost).Methods("POST")
	router.HandleFunc("/instances", HandleIdentityInstancesPost).Methods("POST")
	router.HandleFunc("/identities", HandleOwnIdentities).Methods("GET")
	router.HandleFunc("/identities", HandleOwnIdentitiesPost).Methods("POST")

	go func() {
		// Check that the id url has been set
		if identityURL == "" {
			log.Fatal("Missing id url")
		}

		// Fetch self id
		self, err := FetchSelfIdentity(identityURL)
		if err != nil {
			panic(err)
		}

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
