package main

import (
    "bufio"
    "encoding/json"
    "errors"
    "fmt"
    "math/rand"
    "net"
    "net/http"
    "os"
    "strings"
    "time"
)

const settingsFilename = "links.json"

type ShortLinkData struct {
    ShortLink string `json:"shortLink"`
    LongLink  string `json:"longLink"`
}

type StatisticData struct {
    URL      string `json:"url"`
    ShortURL string `json:"shortURL"`
    IP       string `json:"sourceIP"`
}

type ClickStats struct {
    ClicksCount int
}

var (
    shortLinkData  []ShortLinkData
    clickStatsMap  map[string]*ClickStats
)

func generateShortLink(link string) (string, error) {
    fmt.Println("generateShortLink(", link, ")")
    alphabet := "QWERTYUIOPASDFGHJKLZXCVBNM"
    alphabet = alphabet + strings.ToLower(alphabet) + "1234567890"
    shortLinkChars := ""

    for {
        shortLinkChars = ""
        for i := 0; i < 9; i++ {
            shortLinkChars += string(alphabet[rand.Intn(len(alphabet))])
        }

        _, err := baseFindLink(shortLinkChars)

        fmt.Println("Err:", err.Error())
        if err.Error() == "Link does not exist" {
            break
        }
    }
    fmt.Println("exiting generate with", shortLinkChars)
    return shortLinkChars, nil
}

func baseFindLink(shortLink string) (string, error) {
    fmt.Println("baseFindLink(", shortLink, ")")
    con, err := net.Dial("tcp", "127.0.0.1:6379")

    if err != nil {
        return "", errors.New("Database Unreachable")
    }

    defer con.Close()

    msg := "HGET linksHashtable " + shortLink

    _, err = con.Write([]byte(msg))

    if err != nil {
        return "", err
    }
    reply := make([]byte, 512)

    _, err = con.Read(reply)

    if err != nil {
        return "", err
    }

    cleanReply := strings.TrimSpace(string(reply))
    cleanReply = strings.ReplaceAll(cleanReply, "\n", "")
    fmt.Println(":::::", cleanReply, ":::::")
    if strings.Contains(cleanReply, "not found") {
        return "", errors.New("Link does not exist")
    } else {
        return cleanReply, nil
    }
}

func baseAddLink(shortLink string, longLink string) error {
    fmt.Println("baseAddLink(", shortLink, ",", longLink, ")")
    con, err := net.Dial("tcp", "127.0.0.1:6379")

    if err != nil {
        return errors.New("Database Unreachable")
    }

    defer con.Close()

    msg := "HSET linksHashtable " + shortLink + " " + longLink

    _, err = con.Write([]byte(msg))

    if err != nil {
        return err
    }

    return nil
}

func initializeBase() error {
    con, err := net.Dial("tcp", "127.0.0.1:6379")

    if err != nil {
        return errors.New("Database Unavailable")
    }

    defer con.Close()

    msg := "HSET linksHashtable _test initializationkey"

    _, err = con.Write([]byte(msg))

    if err != nil {
        return err
    }

    return nil
}

func sendStatistics(shortURL, longURL, remoteAddr string) {
    statisticData := StatisticData{
        URL:      longURL,
        ShortURL: shortURL,
        IP:       remoteAddr,
    }

    jsonBytes, err := json.Marshal(statisticData)
    if err != nil {
        fmt.Println("Error marshaling JSON:", err)
        return
    }

    statConn, err := net.Dial("tcp", "localhost:1111")
    if err != nil {
        fmt.Println("Error accepting connection:", err)
        return
    }
    defer statConn.Close()

    _, statErr := statConn.Write(append([]byte("1 "), jsonBytes...))
    if statErr != nil {
        fmt.Println(statErr)
    }
}

func connectionHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        longUrl := r.FormValue("url")
        if longUrl == "" {
            http.Error(w, "Bad Request", http.StatusBadRequest)
            return
        }

        shortURL, _ := generateShortLink(longUrl)

        err := baseAddLink(shortURL, longUrl)

        if err != nil {
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }

        shortLinkData = append(shortLinkData, ShortLinkData{ShortLink: shortURL, LongLink: longUrl})

        err = writeShortLinkDataToFile()
        if err != nil {
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }

        fmt.Fprintf(w, "Shortened URL: 127.0.0.1:8080/%s", shortURL)

        sendStatistics(shortURL, longUrl, r.RemoteAddr)

        // Инициализация счетчика переходов для новой ссылки
        clickStatsMap[shortURL] = &ClickStats{}
    } else if r.Method == http.MethodGet {
        shortUrl := r.URL.Path[1:]

        result, err := baseFindLink(shortUrl)

        fmt.Println("result <<<", result, ">>> error: <<<", err, ">>>")

        if err != nil {
            if err.Error() == "Link does not exist" {
                http.NotFound(w, r)
                return
            } else {
                http.Error(w, "Internal server error"+err.Error(), http.StatusInternalServerError)
                return
            }
        }

        outLink := ""

        if result[0:4] != "http" {
            fmt.Println(result[0:4])
            outLink = "http://" + result
        } else {
            outLink = result
        }

        outLink = strings.ReplaceAll(outLink, "\n", "")
        fmt.Println("outlink <", outLink, ">")
        http.Redirect(w, r, outLink, http.StatusSeeOther)

        // Увеличение счетчика переходов при переходе
        clickStats, exists := clickStatsMap[shortUrl]
        if exists {
            clickStats.ClicksCount++
        }
    } else {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    err := r.ParseForm()
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    str := r.Form["strings"]

    statConn, err := net.Dial("tcp", "localhost:1111")
    defer statConn.Close()
    if err != nil {
        fmt.Println("Error accepting connection:", err)
        os.Exit(1)
    }

    if len(str) == 1 {
        _, statErr := statConn.Write([]byte("2 " + str[0] + "\n"))
        if statErr != nil {
            fmt.Println(statErr)
        }
    } else if len(str) == 2 {
        _, statErr := statConn.Write([]byte("2 " + str[0] + " " + str[1] + "\n"))
        if statErr != nil {
            fmt.Println(statErr)
        }
    } else {
        _, statErr := statConn.Write([]byte("2 " + str[0] + " " + str[1] + " " + str[2] + "\n"))
        if statErr != nil {
            fmt.Println(statErr)
        }
    }

    scanner := bufio.NewScanner(statConn)
    scanner.Scan()
    response := scanner.Text()
    if response == "1" {
        jsonData, jsonErr := os.ReadFile("report.json")
        if err != nil {
            fmt.Println(jsonErr)
        }
        for _, jsonLine := range jsonData {
            fmt.Fprint(w, string(jsonLine))
        }
    }
}

func writeReport(statistics string) {
    file, err := os.Create("report.json")
    if err != nil {
        fmt.Println("Error creating report file:", err)
        return
    }
    defer file.Close()

    _, err = file.WriteString(statistics)
    if err != nil {
        fmt.Println("Error writing to report file:", err)
        return
    }
}

func loadShortLinkDataFromFile() error {
    file, err := os.ReadFile(settingsFilename)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    if len(file) == 0 {
        return nil
    }

    err = json.Unmarshal(file, &shortLinkData)
    if err != nil {
        return err
    }

    return nil
}

func writeShortLinkDataToFile() error {
    jsonData, err := json.MarshalIndent(shortLinkData, "", "  ")
    if err != nil {
        return err
    }

    err = os.WriteFile(settingsFilename, jsonData, 0644)
    if err != nil {
        return err
    }

    return nil
}

func main() {
    rand.Seed(time.Now().UnixNano())
    err := initializeBase()

    if err != nil {
        fmt.Println(err)
        return
    } else {
        fmt.Println("DB accessible!")
    }

    err = loadShortLinkDataFromFile()
    if err != nil {
        fmt.Println("Error loading short link data:", err)
    }

    // Инициализация карты для счетчиков переходов
    clickStatsMap = make(map[string]*ClickStats)

    http.HandleFunc("/", connectionHandler)
    http.HandleFunc("/report", reportHandler)

    fmt.Println("Server started on :8080")
    err = http.ListenAndServe(":8080", nil)
    if err != nil {
        fmt.Println(err)
    }
}
