package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

type Instance struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Identity struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Port     uint   `json:"port"`
	UseSSL   bool   `json:"ssl"`
}

func FetchIdentity(uri string) (identity Identity, err error) {
	fmt.Printf("Trying to fetch %s\n", uri)
	identity = Identity{}
	selfJSON, err := json.Marshal(&rt.self)
	if err == nil {
		resp, err := http.Post(uri, "application/json", bytes.NewBuffer(selfJSON))
		if err == nil {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				fmt.Println(string(body))
				err = json.Unmarshal(body, &identity)
			} else {
				fmt.Println("Got", identity)
			}
		}
	}
	fmt.Println(identity, err)
	return identity, err
}

func (s *Identity) GetURI() string {
	var uri string = ""
	if s.UseSSL == true {
		uri += "https"
	} else {
		uri += "http"
	}
	uri += fmt.Sprintf("://%s:%d", s.Hostname, s.Port)
	return uri
}

func (s *Identity) Send(instance *Instance) (err error) {
	data, err := json.Marshal(&instance)
	if err == nil {
		// var data []byte = []byte(str)
		fmt.Printf("Sending '%s' to %s\n", string(data), s.GetURI())
		_, err = http.Post(fmt.Sprintf("%s/instances", s.GetURI()), "application/json", bytes.NewBuffer(data))
	}
	return err
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
var self Identity = Identity{}
var initURIsString string = ""
var initURIs []string = make([]string, 0)

func init() {
	flag.StringVar(&self.ID, "id", "", "ID")
	flag.StringVar(&self.Hostname, "hostname", "localhost", "Hostname")
	flag.UintVar(&self.Port, "port", 9000, "Port")
	flag.BoolVar(&self.UseSSL, "ssl", false, "SSL")

	flag.StringVar(&initURIsString, "ids", "", "Initial Identities to connect to")
}

func main() {
	// Parse flags
	flag.Parse()

	// Check that hostname and id have been set
	if self.ID == "" {
		log.Fatal("Missing id")
	}
	if self.Hostname == "" {
		log.Fatal("Missing hostname")
	}

	// Show own URI
	fmt.Printf("Starting up as %s\n", self.GetURI())

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

	go func() {
		router := mux.NewRouter().StrictSlash(true)
		// router.HandleFunc("/", Index).Methods("GET")
		router.HandleFunc("/", PostIndex).Methods("POST")
		router.HandleFunc("/instances", PostInstances).Methods("POST")
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", self.Port), router))
	}()
	fmt.Println("Ready...")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case ".":
			break
		case "ls":
			fmt.Printf("Identities connected:\n")
			for _, identity := range rt.identities {
				fmt.Printf(" > %s\n", identity.GetURI())
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
						instance.Type = parts[2]
						instance.Payload = line[len(parts[0])+len(parts[1])+len(parts[2])+3:]
						identity.Send(&instance)
					}
				}
			}
		}
	}
}

// func Index(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("GET /")
// 	json.NewEncoder(w).Encode(self)
// }

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
		json.NewEncoder(w).Encode(self)
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
	// if err := json.Unmarshal(body, &todo); err != nil {
	//     w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//     w.WriteHeader(422) // unprocessable entity
	//     if err := json.NewEncoder(w).Encode(err); err != nil {
	//         panic(err)
	//     }
	// }

	fmt.Println(">>> GOT", string(body))

	json.NewEncoder(w).Encode(self)
}
