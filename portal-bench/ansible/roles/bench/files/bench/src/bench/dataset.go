package bench

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	DataPath = "./data"
	DataSet  BenchDataSet
)

func prepareUserDataSet() {
	log.Println("datapath", DataPath)
	file, err := os.Open(filepath.Join(DataPath, "user.csv"))
	must(err)
	defer file.Close()

	s := bufio.NewScanner(file)
	for i := 0; s.Scan(); i++ {
		line := strings.Split(s.Text(), ",")
		userName := line[0]
		isAdmin := line[1]

		user := &AppUser{
			Name:     userName,
			Password: userName + "201808",
			IsAdmin:  isAdmin,
		}

		DataSet.Users = append(DataSet.Users, user)
	}
}

func PrepareDataSet() {
	prepareUserDataSet()

}
