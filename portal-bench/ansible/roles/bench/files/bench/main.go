package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/benmanns/goworker"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Bench struct {
	Id         int       `gorm:"column:id"`
	Team       string    `gorm:"column:team"`
	Ipaddress  string    `gorm:"column:ipaddress"`
	Result     string    `gorm:"column:result" sql:"type:json"`
	Created_at time.Time `gorm:"column:created_at"`
}

func (t *Bench) TableName() string {
	return "bench"
}

type Results struct {
	Result    bool   `json:"pass"`
	Score     int    `json:"score"`
	Message   string `json:"message"`
	StartTime string `json:"start_time"`
}

func myFunc(queue string, args ...interface{}) error {
	team := fmt.Sprintf("%v", args[0])
	ipaddress := fmt.Sprintf("%v", args[1])

	db, err := gorm.Open("mysql", "hisucon:KCgC6LtWKp5tpKkW#@/hisucon2018_portal?charset=utf8mb4&parseTime=True&loc=Asia%2FTokyo")
	if err != nil {
		log.Fatal("DB connect error.")
		return nil
	}
	defer db.Close()

	now := time.Now()
	layout := "2006-01-02-15:04:05"
	currentDir, _ := os.Getwd()
	resultFile := currentDir + "/logs/" + team + "-" + ipaddress + "." + now.Format(layout) + ".result.json"
	cmd := currentDir + "/bin/bench -remotes=" + ipaddress + " -output " + resultFile
	fmt.Println(cmd)
	err = exec.Command("sh", "-c", cmd).Run()
	result := true
	// quick fix
	bencherror := "{\"job_id\":\"\",\"ip_addrs\":\"xx.xx.xx.xx\",\"pass\":false,\"score\":0,\"message\":\"ベンチマークの実行に失敗しました。再実行を行ってください。\",\"error\":null,\"log\":null,\"load_level\":0,\"start_time\":\"2018-08-30T07:46:57.434507962+09:00\",\"end_time\":\"2018-08-30T07:48:00.192811932+09:00\"}"
	if err != nil {
		fmt.Println("benchmark execute error.", err)
		result = false
	}

	if result {
		jsonResult, _ := ioutil.ReadFile(resultFile)
		var result = Bench{Team: team, Ipaddress: ipaddress, Result: string(jsonResult)}
		db.Create(&result)

	} else {
		var result = Bench{Team: team, Ipaddress: ipaddress, Result: bencherror}
		db.Create(&result)

	}

	fmt.Println(queue, args[0], args[1])
	time.Sleep(10 * time.Second)
	return nil
}

func init() {
	settings := goworker.WorkerSettings{
		URI:            "redis://localhost:6379/",
		Connections:    2,
		Queues:         []string{"myqueue", "delimited", "queues"},
		UseNumber:      true,
		ExitOnComplete: false,
		Concurrency:    1,
		Namespace:      "resque:",
		IntervalFloat:  1.0,
	}
	goworker.SetSettings(settings)
	goworker.Register("Hisucon2018", myFunc)
}

func main() {
	if err := goworker.Work(); err != nil {
		fmt.Println("Error:", err)
	}
}
