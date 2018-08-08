package provenance

import (
	"encoding/json"
	"net/http"
	"strings"
	//"log"
	"os"
	//"log"
	"fmt"
	//"io/ioutil"

	//      "strings"
	//      "io/ioutil"
	//      "log"
	//      "crypto/tls"
	//      "context"
	//      "gopkg.in/yaml.v2"
	"bufio"
	//cert "crypto/x509"
	//"net/http"
	//"net/url"

	"k8s.io/apiserver/pkg/apis/audit/v1beta1"
)

type RequestObject struct {
}

var (
	serviceHost    string
	servicePort    string
	Namespace      string
	httpMethod     string
	etcdServiceURL string
	ETCD_CLUSTER   string
	// ObjectFullProvenances        []Object
	ObjectFullProvenance Object
	debug                bool
)

//if there is more than one postgres instance, need to generalize w a map
//for more than one postgres instance, need to generalize with a map

type Event v1beta1.Event

//for example a postgres
type Object map[int]Spec
type Spec struct {
	attributeToData map[string]string
}

func main() {
	CollectProvenance()
}

func init() {
	serviceHost = os.Getenv("KUBERNETES_SERVICE_HOST")
	servicePort = os.Getenv("KUBERNETES_SERVICE_PORT")
	Namespace = "default"
	httpMethod = http.MethodGet
	etcdServiceURL = "http://example-etcd-cluster-client:2379"
	ETCD_CLUSTER = "EtcdCluster"
	ObjectFullProvenance = make(map[int]Spec) //need to generalize for other ObjectFullProvenances
	debug = true
}
func CollectProvenance() {
	// for {
	parse()
	// fmt.Println(ObjectFullProvenance.String())
	fmt.Println(ObjectFullProvenance.FieldDiff("users", 3, 4))
	// time.Sleep(time.Second * 5)
	// }
	//	done <- true
}
func NewSpec() *Spec {
	var s Spec
	s.attributeToData = make(map[string]string)
	return &s
}

func (s *Spec) String() string {
	var b strings.Builder
	for attribute, data := range s.attributeToData {
		fmt.Fprintf(&b, "Attribute: %s Data: %s\n", attribute, data)
	}
	return b.String()
}

func (o *Object) String() string {
	var b strings.Builder
	for version, spec := range *o {
		fmt.Fprintf(&b, "Version: %d Data: %s\n", version, spec.String())
	}
	return b.String()
}
func (o *Object) LatestVersion(vNum int) int {
	return len(ObjectFullProvenance)
}
func (o *Object) Version(vNum int) Spec {
	return ObjectFullProvenance[vNum]
}

//add type of ObjectFullProvenance, postgreses for example
func (o *Object) SpecHistory() []Spec {
	s := make([]Spec, 0)
	for _, spec := range ObjectFullProvenance {
		s = append(s, spec)
	}
	return s
}

//add type of ObjectFullProvenance, postgreses for example
func (o *Object) SpecHistoryInterval(vNumStart, vNumEnd int) []Spec {
	s := make([]Spec, 0)
	for key, spec := range ObjectFullProvenance {
		if key >= vNumStart && key <= vNumEnd {
			s = append(s, spec)
		}
	}
	return s
}

//add type of ObjectFullProvenance, postgreses for example
func (o *Object) FullDiff(vNumStart, vNumEnd int) string {
	var b strings.Builder
	sp1 := ObjectFullProvenance[vNumStart]
	sp2 := ObjectFullProvenance[vNumEnd]
	for attribute, data1 := range sp1.attributeToData {
		if data2, ok := sp2.attributeToData[attribute]; ok {
			if data1 != data2 {
				fmt.Fprintf(&b, "FOUND DIFF")
				fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumStart, data1)
				fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumEnd, data2)
			} else {
				fmt.Fprintf(&b, "No difference for attribute %s \n", attribute)
			}
		} else {
			fmt.Fprintf(&b, "FOUND DIFF")
			fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumStart, data1)
			fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumEnd, "No attribute found.")
		}
	}
	return b.String()
}

//add type of ObjectFullProvenance, postgreses for example
func (o *Object) FieldDiff(fieldName string, vNumStart, vNumEnd int) string {
	var b strings.Builder
	data1, ok1 := ObjectFullProvenance[vNumStart].attributeToData[fieldName]
	data2, ok2 := ObjectFullProvenance[vNumEnd].attributeToData[fieldName]
	switch {
	case ok1 && ok2:
		if data1 != data2 {
			fmt.Fprintf(&b, "FOUND DIFF\n")
			fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumStart, data1)
			fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumEnd, data2)
		} else {
			fmt.Fprintf(&b, "No difference for attribute %s \n", fieldName)
		}
	case !ok1 && ok2:
		fmt.Fprintf(&b, "FOUND DIFF")
		fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumStart, "No attribute found.")
		fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumEnd, data2)
	case ok1 && !ok2:
		fmt.Fprintf(&b, "FOUND DIFF")
		fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumStart, data1)
		fmt.Fprintf(&b, "Spec version %d:\n %s\n", vNumEnd, "No attribute found.")
	case !ok1 && !ok2:
		fmt.Fprintf(&b, "Attribute not found in either version %d or %d", vNumStart, vNumEnd)
	}
	return b.String()
}

//Ref:https://www.sohamkamani.com/blog/2017/10/18/parsing-json-in-golang/#unstructured-data
func parse() {
	fmt.Println("PARSING")

	if _, err := os.Stat("kube-apiserver-audit.log"); os.IsNotExist(err) {
		if debug {
			fmt.Println(fmt.Sprintf("could not stat the path %s", err))
		}
		panic(err)
	}
	log, err := os.Open("/tmp/kube-apiserver-audit.log")
	if err != nil {
		if debug {
			fmt.Println(fmt.Sprintf("could not open the log file %s", err))
		}
		panic(err)
	}
	defer log.Close()

	scanner := bufio.NewScanner(log)
	for scanner.Scan() {

		eventJson := scanner.Bytes()

		var event Event
		err := json.Unmarshal(eventJson, &event)
		if err != nil {
			s := fmt.Sprintf("Problem parsing event ObjectFullProvenance's json %s", err)
			fmt.Println(s)
		}

		requestobj := event.RequestObject

		ParseRequestObject(requestobj.Raw)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println("DONE PARSING")
	fmt.Println("")

}

func ParseRequestObject(requestObjBytes []byte) {
	fmt.Println("entering parse request")

	var result map[string]interface{}
	json.Unmarshal([]byte(requestObjBytes), &result)

	l1, ok := result["metadata"].(map[string]interface{})
	if !ok {
		sp, _ := result["spec"].(map[string]interface{})
		//TODO: for the case where a crd ObjectFullProvenance is first created, like initialize,
		//the metadata spec is empty. instead the spec field has the data
		fmt.Println(sp)
		return
	}
	l2, ok := l1["annotations"].(map[string]interface{})
	l3, ok := l2["kubectl.kubernetes.io/last-applied-configuration"].(string)
	if !ok {
		fmt.Println("Incorrect parsing of the auditEvent.requestObj.metadata")
	}
	in := []byte(l3)
	var raw map[string]interface{}
	json.Unmarshal(in, &raw)
	spec, ok := raw["spec"].(map[string]interface{})

	if ok {
		fmt.Println("Successfully parsed")
	}

	saveProvenance(spec)

	fmt.Println("exiting parse request")
}
func saveProvenance(spec map[string]interface{}) {
	mySpec := *NewSpec()
	newVersion := 1 + len(ObjectFullProvenance)
	for attribute, value := range spec {
		bytes, err := json.MarshalIndent(value, "", "    ")
		if err != nil {
			fmt.Println("Error could not marshal json")
		}
		attributeData := string(bytes)
		mySpec.attributeToData[attribute] = attributeData
	}
	ObjectFullProvenance[newVersion] = mySpec
}
