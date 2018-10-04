package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
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

func checkIPaddressFormat(ipaddress string) bool {
	cidr := "192.168.0.0/24"
	_, ipnet, _ := net.ParseCIDR(cidr)

	return ipnet.Contains(net.ParseIP(ipaddress))
}

func checkHostUp(ipaddress string) bool {
	res := true
	result, _ := exec.Command("sh", "-c", "nmap -sP "+ipaddress+" | grep \"Host is up\"").Output()
	if string(result) == "" {
		log.Println(ipaddress + "is not up.")
		res = false
	}
	return res
}

func main() {
	router := gin.Default()
	currentDir, _ := os.Getwd()
	router.LoadHTMLGlob(currentDir + "/templates/*.tmpl")

	router.GET("/top/:team/:ipaddress", func(c *gin.Context) {
		team := c.Param("team")
		ipaddress := c.Param("ipaddress")
		checkIPaddressFormat(ipaddress)

		if checkIPaddressFormat(ipaddress) {
			if !checkHostUp(ipaddress) {
				c.HTML(http.StatusOK, "error.tmpl", gin.H{
					"message": ipaddress + "は Host Up してません。",
				})
				return
			}
		} else {
			c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{
				"message": ipaddress + "は不正な値です。",
			})
			return
		}

		db, err := gorm.Open("mysql", "hisucon:KCgC6LtWKp5tpKkW#@/hisucon2018_portal?charset=utf8mb4&parseTime=True")
		if err != nil {
			log.Fatal("DB connect error.", err)
			return
		}
		defer db.Close()

		var bench []Bench
		db.Select("result").Where("team = ? AND ipaddress = ?", team, ipaddress).Order("created_at DESC").Find(&bench)

		var results []Results
		for _, res := range bench {
			var result Results
			str := fmt.Sprintf("%v", res.Result)
			if err := json.Unmarshal([]byte(str), &result); err != nil {
				log.Fatal(err)
			}
			results = append(results, result)
		}

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"results": results,
			"url":     "/bench/" + team + "/" + ipaddress,
		})
	})

	router.GET("/bench/:team/:ipaddress", func(c *gin.Context) {
		team := c.Param("team")
		ipaddress := c.Param("ipaddress")
		checkIPaddressFormat(ipaddress)

		if checkIPaddressFormat(ipaddress) {
			if !checkHostUp(ipaddress) {
				c.HTML(http.StatusOK, "error.tmpl", gin.H{
					"message": ipaddress + "は Host Up してません。",
				})
				return
			}
		} else {
			c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{
				"message": ipaddress + "は不正な値です。",
			})
			return
		}

		cmd := "redis-cli RPUSH resque:queue:myqueue '{\"class\":\"Hisucon2018\",\"args\":[\"" + team + "\",\"" + ipaddress + "\"]}'"
		err := exec.Command("sh", "-c", cmd).Run()
		if err != nil {
			fmt.Println("benchmark execute error.", err)
			c.JSON(412, string("処理に失敗しました。"))

		} else {
			c.JSON(200, string("ok"))
		}

	})

	router.Run(":80")

}
