package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Schedule struct {
	Sched []struct {
		DatePair     string `json:"datePair"`
		DayWeek      string `json:"dayWeek"`
		MainSchedule []struct {
			Subject   string `json:"SubjSN"`
			FullName  string `json:"FIO"`
			LoadKind  string `json:"LoadKindSN"`
			TimeStart string `json:"TimeStart"`
			Aud       string `json:"Aud"`
		} `json:"mainSchedule"`
	} `json:"Sched"`
}

func getScheduleData(groupId, yearId, mountNum int, schedChan chan *Schedule) {
	var sched *Schedule
	url := fmt.Sprintf(
		"https://www.ursei.ac.ru/Services/GetGsSched?grpid=%d&yearid=%d&monthnum=%d",
		groupId, yearId, mountNum)

	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err := json.Unmarshal([]byte(body), &sched); err != nil {
		log.Println(err)
	}

	schedChan <- sched
}

func getAllSchedules() {
	groups := []int{26066, 26067}
	schedChan := make(chan *Schedule, len(groups))

	for _, group := range groups {
		go getScheduleData(group, 26, 12, schedChan)
	}

	for i := 0; i < len(groups); i++ {
		sched :=<-schedChan
		sched.printSched()
	}
}

func (sched *Schedule) printSched() {
	for _, day := range sched.Sched {
		fmt.Printf("Дата: %v, День: %v\n", day.DatePair, day.DayWeek)
		fmt.Printf("Расписание:\n")
		for _, pair := range day.MainSchedule {
			fmt.Printf(
				"Предмет: %v, Преподаватель: %v, Тип занятия: %v, Время начала: %v, Аудитория: %v\n",
				pair.Subject, pair.FullName, pair.LoadKind, pair.TimeStart, pair.Aud)
		}
		fmt.Println()
	}
}

func main() {
	getAllSchedules()
}
