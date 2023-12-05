package main
 
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)
 
type connectionReport struct {
	ShortUrl string `json:"shortURL"`
	OutLink  string `json:"outLink"`
	Host     string `json:"originHost"`
}
 
type JSONEntry struct {
	ID       int    `json:"id"`
	PID      int    `json:"pid"`
	URL      string `json:"url"`
	ShortURL string `json:"shortURL"`
	SourceIP string `json:"sourceIP"`
	Time     string `json:"time"`
	Count    int    `json:"count"`
}
 
type Payload struct {
	Dimensions []string `json:"Dimensions"`
}
 
func newRedirectHandler(w http.ResponseWriter, r *http.Request) {
	var reportData connectionReport
	err := json.NewDecoder(r.Body).Decode(&reportData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
 
	statConnections(reportData.OutLink, reportData.ShortUrl, reportData.Host)

 
	return
}
func reportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
 
	var payload Payload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
 
	fmt.Println("Received dimensions:", payload.Dimensions)
 
	response, err := os.ReadFile("connections.json")
	if err != nil {
		fmt.Println("Ошибка при чтении json-файла:", err)
		return
	}
 
	JsonFile := ByteToJSON(response)
 
	jsonData := createReport(payload.Dimensions, JsonFile)
 
	err = writeJSONToFile(jsonData, "report.json")
	if err != nil {
		fmt.Println("Ошибка записи в файл:", err)
		return
	}
 
	data, err := ioutil.ReadFile("report.json")
 
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
 
	reportContent := string(data)
 
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(reportContent))
}
 
func main() {
	rand.Seed(time.Now().UnixNano())
 
	fmt.Println("Stats server up at 127.0.0.1:6565")
 
	http.HandleFunc("/", newRedirectHandler)
	http.HandleFunc("/report", reportHandler)
 
	log.Fatal(http.ListenAndServe(":6565", nil))
}
 
func statConnections(url, shortURL, ip string) {
 
	parentConnect := JSONEntry{
		URL:      url,
		ShortURL: shortURL,
		Count:    1,
	}
 
	newConnect := JSONEntry{
		SourceIP: ip,
		Time:     time.Now().Format("2006-01-02 15:04"),
		Count:    1,
	}
 
	connections, err := readConnectionsFromFile()
	if err != nil {
		fmt.Println("Ошибка чтения из файла:", err)
		return
	}
 
	if connections == nil {
		connections = []JSONEntry{}
	}
 
	parentConnect.ID = generateUniqueID(connections)
	if UniqueParents(connections, parentConnect.URL) == true {
		connections = append(connections, parentConnect)
	} else {
		ParentsCount(connections, parentConnect.URL)
	}
 
	newConnect.ID = generateUniqueID(connections)
	newConnect.PID = generatePID(connections, url)
	connections = append(connections, newConnect)
 
	err = writeConnectionsToFile(connections)
	if err != nil {
		fmt.Println("Ошибка записи в файл:", err)
		return
	}
 
}
 
func ByteToJSON(file []byte) []JSONEntry {
	var Connections []JSONEntry
 
	if len(file) == 0 {
		return nil
	}
 
	err := json.Unmarshal(file, &Connections)
	if err != nil {
		return nil
	}
 
	return Connections
}
 
func readConnectionsFromFile() ([]JSONEntry, error) {
	var Connections []JSONEntry
 
	file, err := os.ReadFile("connections.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
 
	if len(file) == 0 {
		return nil, nil
	}
 
	err = json.Unmarshal(file, &Connections)
	if err != nil {
		return nil, err
	}
 
	return Connections, nil
}
 
func writeConnectionsToFile(workers []JSONEntry) error {
	jsonData, err := json.MarshalIndent(workers, "", "  ")
	if err != nil {
		return err
	}
 
	err = os.WriteFile("connections.json", jsonData, 0644)
	if err != nil {
		return err
	}
 
	return nil
}
 
func UniqueParents(Connections []JSONEntry, url string) bool {
	for _, connect := range Connections {
		if connect.URL == url {
			return false
		}
	}
	return true
}
 
func ParentsCount(Connections []JSONEntry, url string) {
	for index := range Connections {
		if Connections[index].URL == url {
			Connections[index].Count++
			return
		}
	}
}
 
func generateUniqueID(Connections []JSONEntry) int {
	maxID := 0
	for _, connect := range Connections {
		if connect.ID > maxID {
			maxID = connect.ID
		}
	}
	return maxID + 1
}
 
func generatePID(Connections []JSONEntry, url string) int {
	PID := 0
	for _, connect := range Connections {
		if connect.URL == url {
			PID = connect.ID
		}
	}
	return PID
}
 
func writeJSONToFile(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
 
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
 
	if err := encoder.Encode(data); err != nil {
		return err
	}
 
	return nil
}
 
func findURLByID(id int, connections []JSONEntry) string {
	for _, conn := range connections {
		if conn.ID == id {
			return conn.URL
		}
	}
	return ""
}
 
func findShortURLByID(id int, connections []JSONEntry) string {
	for _, conn := range connections {
		if conn.ID == id {
			return conn.ShortURL
		}
	}
	return ""
}
func createReport(detailing []string, connections []JSONEntry) map[string]interface{} {
	report := make(map[string]interface{})
 
	for _, connection := range connections {
		if connection.PID == 0 {
			continue
		}
 
		ip := connection.SourceIP
		Time := connection.Time[11:]
		shortURL := findShortURLByID(connection.PID, connections)
		url := findURLByID(connection.PID, connections) + " (" + shortURL + ")"
 
		currLevel := report
		for _, level := range detailing {
 
			if level == "SourceIP" {
				if _, ok := currLevel[ip]; !ok {
					currLevel[ip] = make(map[string]interface{})
					if _, ok := currLevel["Sum"]; !ok {
						currLevel["Sum"] = 0
					}
				}
				currLevel = currLevel[ip].(map[string]interface{})
			} else if level == "TimeInterval" {
				if _, ok := currLevel[Time]; !ok {
					currLevel[Time] = make(map[string]interface{})
					if _, ok := currLevel["Sum"]; !ok {
						currLevel["Sum"] = 0
					}
				}
				currLevel = currLevel[Time].(map[string]interface{})
			} else if level == "URL" {
				if _, ok := currLevel[url]; !ok {
					currLevel[url] = make(map[string]interface{})
					if _, ok := currLevel["Sum"]; !ok {
						currLevel["Sum"] = 0
					}
				}
				currLevel = currLevel[url].(map[string]interface{})
			}
 
			if _, ok := currLevel["Sum"]; !ok {
				currLevel["Sum"] = 0
			}
			currLevel["Sum"] = currLevel["Sum"].(int) + 1
		}
	}
 
	delete(report, "Sum")
 
	return report
}