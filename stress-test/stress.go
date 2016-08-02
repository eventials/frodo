package main

import (
    "bufio"
    "net/http"
    "time"
    "fmt"
    "os"
    "strconv"
)


func main() {
    args := os.Args[1:]
    var url string = "http://local.eventials.com:3000/talk-state/123456"
    var qtd int = 10

    if len(args) > 0 {
        url = args[0]       
    }

    if len(args) > 1 {
        var err error
        qtd, err = strconv.Atoi(args[1])
        if err != nil {
            panic(err)
        }
    }

    fmt.Printf("QTD: %d URL: %s\n", qtd, url)
    for i := 0; i < qtd; i++ {
        go func(){
            client := &http.Client{
            }
            req, _ := http.NewRequest("GET", url, nil)
            req.Header.Set("Connection", "keep-alive")

            res, err := client.Do(req)
            if err != nil {
                fmt.Println(err)
            } else {
                br := bufio.NewReader(res.Body)
                defer res.Body.Close()
                for {
                    content, err := br.ReadBytes('\n')
                    if err != nil {
                        fmt.Println(err)
                    } else {
                        fmt.Println(string(content))                            
                    }
                }
            }
        }()
        time.Sleep(15 * time.Millisecond)
    }

    select {}
 }
