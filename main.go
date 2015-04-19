package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	flag.StringVar(&initURIsString, "ids", "", "Initial Identities to connect to")
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
	router.HandleFunc("/", Index).Methods("GET")
	router.HandleFunc("/", PostIndex).Methods("POST")
	router.HandleFunc("/instances", PostInstances).Methods("POST")

	if os.Getenv("PORT") != "" {
		tempLocalPort, _ := strconv.Atoi(os.Getenv("PORT"))
		localPort = uint(tempLocalPort)
		fmt.Println("Fast forwarding HTTP server")
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", localPort), router))
	}

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

	if initURIsString != "" {
		initURIs = strings.Split(initURIsString, ",")
	}

	rt = newRoutingTable(&self)
	if len(initURIs) > 0 {
		for _, uri := range initURIs {
			var identity Identity
			identity, err := FetchIdentity(uri)
			if err != nil {
				fmt.Printf("Could not fetch %s, error: %s\n", uri, err)
			} else {
				rt.insertIdentity(&identity)
			}
		}
	}

	if localPort == 0 {
		localPort = self.Port
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", localPort), router))
	}
	fmt.Println("Ready...")

	if os.Getenv("ENV") != "DEV" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			switch line {
			case ".":
				break
			case "ls":
				fmt.Printf("Identities connected:\n")
				for _, identity := range rt.identities {
					fmt.Printf(" > %s %s\n", identity.GetURI(), identity.ID)
				}
			default:
				var parts []string = strings.Split(line, " ")
				if len(parts) > 3 {
					if parts[0] == "send" {
						identity, err := rt.Get(parts[1])
						if err != nil {
							fmt.Println(err)
						} else {
							var instance Instance = Instance{}
							instance.Owner = rt.self
							instance.Payload.Owner = rt.self.ID
							instance.Payload.Schema = parts[2]
							instance.Payload.Data = line[len(parts[0])+len(parts[1])+len(parts[2])+3:]
							instance.Sign()
							identity.Send(&instance)
						}
					}
				}
			}
		}
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET /")
	json.NewEncoder(w).Encode(rt.self)
}

func PostIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /")

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	var identity Identity = Identity{}
	if err := json.Unmarshal(body, &identity); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	} else {
		rt.insertIdentity(&identity)
		json.NewEncoder(w).Encode(rt.self)
	}
}

func PostInstances(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /instances")

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	var instance Instance = Instance{}
	instance.SetPayloadFromJson(body)

	body, err = instance.ToJSON()
	if err == nil {
		fmt.Println(">>> GOT", string(body))
		valid, err := instance.Verify()
		if err == nil {
			if valid == true {
				fmt.Println(">>> IS VALID")
			} else {
				fmt.Println(">>> IS *NOT* VALID")
			}
		} else {
			fmt.Println(">>> error validating", err)
		}
	}
	json.NewEncoder(w).Encode(self)
}
