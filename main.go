package main

import (
	"encoding/json"
	"fmt"
	"io"
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

func main() {
	var sched Schedule
	resp, err := http.Get("https://www.ursei.ac.ru/Services/GetGsSched?grpid=26066&yearid=26&monthnum=12")
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err := json.Unmarshal([]byte(body), &sched); err != nil {
		fmt.Println(err)
	}

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
