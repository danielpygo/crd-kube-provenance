package provenance

import (
	"encoding/json"
	"os"
	"strings"
	//"log"
	"time"
	"fmt"
	"io/ioutil"
	//      "strings"
	//      "io/ioutil"
	//      "log"
	//      "crypto/tls"
	//      "context"
	//      "gopkg.in/yaml.v2"
	"bufio"
	"net/http"
	cert "crypto/x509"
	"k8s.io/apiserver/pkg/apis/audit/v1beta1"
)
type User struct {
	Password string
	Username string
}

type Spec struct {
	Users     []User
	Databases []string
}
type RequestObject struct {
	Specs Spec
}

var (
	serviceHost    string
	servicePort    string
	Namespace      string
	httpMethod     string
	etcdServiceURL string
	ETCD_CLUSTER   string
	provenance     ProvenanceInfo
	debug          bool
)

//if there is more than one postgres instance, need to generalize w a map
//for more than one postgres instance, need to generalize with a map
type ProvenanceInfo struct {
	UserToPassword        map[string]string
	UserToPasswordChanged map[string]int
	Databases             []string
	NumDatabases          int
	NumUsers              int
}

type Event v1beta1.Event

func init() {
	serviceHost = os.Getenv("KUBERNETES_SERVICE_HOST")
	servicePort = os.Getenv("KUBERNETES_SERVICE_PORT")
	Namespace = "default"
	httpMethod = http.MethodGet
	etcdServiceURL = "http://example-etcd-cluster-client:2379"
	ETCD_CLUSTER = "EtcdCluster"
	provenance = ProvenanceInfo{
		UserToPassword:        make(map[string]string),
		UserToPasswordChanged: make(map[string]int),
		Databases:             make([]string, 0),
		NumDatabases:          0,
		NumUsers:              0}
	debug = true
}
func CollectProvenance(done chan bool) {
	      for {
		//requestObjects := parse(done)
	//saveProvenanceInformation(requestObjects)
		fmt.Println("collecting")
	        time.Sleep(time.Second * 5 )
	      }
//	done <- true
}

//change to save in etcd pod
//need to change to update based on stream of events
func saveProvenanceInformation(requestObjects []RequestObject) {
	for _, obj := range requestObjects {
		dbCount := provenance.NumDatabases
		userCount := provenance.NumDatabases
		reqDBCount := len(obj.Specs.Databases)
		reqUserCount := len(obj.Specs.Users)
		var typeOfRequest string
		//the json object gives no indication as per what
		//yaml request was called so have to figure it out based
		//on the spec
		switch {
		case dbCount == 0 && userCount == 0:
			typeOfRequest = "initialize-db"
		case dbCount < reqDBCount && reqUserCount == 0:
			typeOfRequest = "add-db"
		case dbCount > reqDBCount && reqUserCount == 0:
			typeOfRequest = "delete-db"
		case reqDBCount == 0 && userCount < reqUserCount:
			typeOfRequest = "add-user"
		case reqDBCount == 0 && userCount > reqUserCount:
			typeOfRequest = "delete-user"
		case dbCount == reqDBCount && userCount == reqUserCount:
			typeOfRequest = "modify-password"
		}
		switch typeOfRequest {
		case "initialize-db":
			for _, database := range obj.Specs.Databases {
				provenance.Databases = append(provenance.Databases, database)
			}
			for _, user := range obj.Specs.Users {
				provenance.UserToPassword[user.Username] = user.Password
			}
		case "delete-db", "add-db":
			provenance.Databases = obj.Specs.Databases
		case "add-user":
			for _, user := range obj.Specs.Users {
				forAdd := false
				if _, ok := provenance.UserToPassword[user.Username]; !ok {
					forAdd = true
				}
				if forAdd {
					provenance.UserToPassword[user.Username] = user.Password
					provenance.UserToPasswordChanged[user.Username] = 0
				}
			}
		case "delete-user":
			for key, _ := range provenance.UserToPassword {
				forDelete := true
				for _, user := range obj.Specs.Users {
					if key == user.Username {
						forDelete = false
					}
				}
				if forDelete {
					delete(provenance.UserToPassword, key)
					delete(provenance.UserToPasswordChanged, key)
				}
			}
		case "modify-password":
			for key, value := range provenance.UserToPassword {
				for _, user := range obj.Specs.Users {
					if key == user.Username && value != user.Password {
						provenance.UserToPassword[key] = user.Password
						provenance.UserToPasswordChanged[key] += 1
					}
				}
			}
		}
		provenance.NumDatabases = len(provenance.Databases)
		provenance.NumUsers = len(provenance.UserToPassword)
	}
}
func (r *RequestObject) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Users: [")
	for _, user := range r.Specs.Users {
		fmt.Fprintf(&b, "{%s %s}\n", user.Username, user.Password)
	}
	fmt.Fprintf(&b, "]\n")

	fmt.Fprintf(&b, "Databases: [")
	for _, database := range r.Specs.Databases {
		fmt.Fprintf(&b, " {%s}\n", database)
	}
	fmt.Fprintf(&b, "]")
	return b.String()
}

func (p *ProvenanceInfo) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Provenance: {\n")
	fmt.Fprintf(&b, "Users: [\n")
	for key, value := range p.UserToPassword {
		fmt.Fprintf(&b, "\t{%s %s}\n", key, value)
	}
	fmt.Fprintf(&b, "\t]\n")

	fmt.Fprintf(&b, "Databases: [\n")
	for _, database := range p.Databases {
		fmt.Fprintf(&b, "\t{%s}\n", database)
	}
	fmt.Fprintf(&b, "\t]\n")
	fmt.Fprintf(&b, "Number of Databases: %d\n", p.NumDatabases)
	fmt.Fprintf(&b, "Number of Users: %d\n}", p.NumUsers)

	return b.String()
}

//Ref:https://www.sohamkamani.com/blog/2017/10/18/parsing-json-in-golang/#unstructured-data
func parse(done chan bool) []RequestObject {
	f, err := os.Create("/tmp/crdprovenance-output.txt")
	defer f.Close()
	w := bufio.NewWriter(f)
	w.WriteString("Output of crdprovenance\n")

	if _, err := os.Stat("/tmp/kube-apiserver-audit.log"); os.IsNotExist(err) {
		if debug {
			w.WriteString(fmt.Sprintf("could not stat the path %s", err))
		}
		done <- true
		panic(err)
	}
	log, err := os.Open("/tmp/kube-apiserver-audit.log")
	if err != nil {
		if debug {
			w.WriteString(fmt.Sprintf("could not open the log file %s", err))
		}
		panic(err)
		done <- true
	}
	defer log.Close()

	// var events []Event
	var reqs []RequestObject

	scanner := bufio.NewScanner(log)
	for scanner.Scan() {
		eventJson := scanner.Bytes()

		var event Event
		err := json.Unmarshal(eventJson, &event)
		if err != nil {
			s := fmt.Sprintf("Problem parsing event object's json %s", err)
			w.WriteString(s)
		}

		requestobj := event.RequestObject

		req := ParseRequestObject(requestobj.Raw)
		reqs = append(reqs, req)
		if debug {
			w.WriteString("******************************\n")
			requestJSON, err := json.MarshalIndent(req, "", "    ")

			if err != nil {
				w.WriteString(fmt.Sprintf("error parsing into Event struct %s", err))
			}
			// events = append(events, event)

			eventJSON, err := json.MarshalIndent(event, "", "    ")
			if err != nil {
				w.WriteString("error parsing into Event struct")
			}
			w.Write(requestJSON)
			w.Write(eventJSON)
			w.WriteString("******************************\n")
			w.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return reqs
}

func ParseRequestObject(requestObjBytes []byte) RequestObject {
	fmt.Println("entering parse request")

	var result map[string]interface{}
	json.Unmarshal([]byte(requestObjBytes), &result)

	specs, ok := result["spec"].(map[string]interface{})
	parsedRequest := RequestObject{}
	if ok {
		for key, value := range specs {
			if key == "users" {
				spec := value.([]interface{})
				users := make([]User, 0)
				for _, val := range spec {
					user := User{}
					usrEntry := val.(map[string]interface{})
					user.Username = usrEntry["username"].(string)
					user.Password = usrEntry["password"].(string)
					users = append(users, user)
				}
				parsedRequest.Specs.Users = users
			}
			if key == "databases" {
				spec := value.([]interface{})
				dbs := make([]string, 0)
				for _, value := range spec {
					dbs = append(dbs, value.(string))
				}
				parsedRequest.Specs.Databases = dbs
			}
		}
	}
	fmt.Println("exiting parse request")
	return parsedRequest
}

//func getResourceListContent(resourceApiVersion, resourcePlural string) []byte {
	//fmt.Println("Entering getResourceListContent")
//        f, err := os.Create("/tmp/provenance-output.txt")
//	log.SetOutput(f)
//	log.Println("hello")
//	url1 := fmt.Sprintf("https://%s:%s/%s/namespaces/%s/%s", serviceHost, servicePort, resourceApiVersion, Namespace, resourcePlural)
	//fmt.Printf("Url:%s\n",url1)
//	caToken := getToken()
//	caCertPool := getCACert()
//	u, err := url.Parse(url1)
//	if err != nil {
//		panic(err)
//	}
//	req, err := http.NewRequest(httpMethod, u.String(), nil)
//	if err != nil {
//		fmt.Println(err)
//	}
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(caToken)))
//	client := &http.Client{
//		Transport: &http.Transport{
//			TLSClientConfig: &tls.Config{
//				RootCAs: caCertPool,
//			},
//		},
//	}
//	resp, err := client.Do(req)
//	if err != nil {
//		log.Printf("sending request failed: %s", err.Error())
//		fmt.Println(err)
//	}
//	defer resp.Body.Close()
//	resp_body, _ := ioutil.ReadAll(resp.Body)
//
//	//fmt.Println(resp.Status)
	//fmt.Println(string(resp_body))
	//fmt.Println("Exiting getResourceListContent")
//	return resp_body
//}


// Ref:https://stackoverflow.com/questions/30690186/how-do-i-access-the-kubernetes-api-from-within-a-pod-container
func getToken() []byte {
	caToken, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		panic(err) // cannot find token file
	}
	//fmt.Printf("Token:%s", caToken)
	return caToken
}

// Ref:https://stackoverflow.com/questions/30690186/how-do-i-access-the-kubernetes-api-from-within-a-pod-container
func getCACert() *cert.CertPool {
	caCertPool := cert.NewCertPool()
	caCert, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		panic(err) // Can't find cert file
	}
	//fmt.Printf("CaCert:%s",caCert)
	caCertPool.AppendCertsFromPEM(caCert)
	return caCertPool
}

