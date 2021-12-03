package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

type ScheduleIn struct {
	Sched []struct {
		DatePair     string `json:"datePair"`
		DayWeek      string `json:"dayWeek"`
		MainSchedule []struct {
			SubjSN     string `json:"SubjSN"`
			FIO        string `json:"FIO"`
			LoadKindSN string `json:"LoadKindSN"`
			TimeStart  string `json:"TimeStart"`
			Aud        string `json:"Aud"`
		} `json:"mainSchedule"`
	} `json:"Sched"`
}

type Pair struct {
	Subject   string `json:"subject"`
	Teacher   string `json:"teacher"`
	ClassType string `json:"classType"`
	TimeStart string `json:"timeStart"`
	Audience  string `json:"audience"`
}

type DaySchedule struct {
	Date      string `json:"date"`
	DayOfWeek string `json:"dayOfWeek"`
	Pairs     []Pair `json:"pairs"`
}

type MonthSchedule struct {
	GroupId int           `json:"group"`
	Month   int           `json:"month"`
	YearId  int           `json:"yearId"`
	Days    []DaySchedule `json:"days"`
}

var redisConnection = redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
}

func (scheduleIn *ScheduleIn) convertFormat(groupId, yearId, monthNum int) *MonthSchedule {
	var monthSchedule MonthSchedule

	monthSchedule.GroupId = groupId
	monthSchedule.Month = monthNum
	monthSchedule.YearId = yearId

	for _, dayIn := range scheduleIn.Sched {
		var daySchedule DaySchedule
		daySchedule.Date = dayIn.DatePair
		daySchedule.DayOfWeek = dayIn.DayWeek
		for _, pairIn := range dayIn.MainSchedule {
			var pair Pair

			pair.Subject = pairIn.SubjSN
			pair.Teacher = pairIn.FIO
			pair.ClassType = pairIn.LoadKindSN
			pair.TimeStart = pairIn.TimeStart
			pair.Audience = pairIn.Aud

			daySchedule.Pairs = append(daySchedule.Pairs, pair)
		}
		monthSchedule.Days = append(monthSchedule.Days, daySchedule)
	}

	return &monthSchedule
}

func getScheduleData(groupId, yearId, mountNum int) (*MonthSchedule, error) {
	var scheduleIn ScheduleIn
	url := fmt.Sprintf(
		"https://www.ursei.ac.ru/Services/GetGsSched?grpid=%d&yearid=%d&monthnum=%d",
		groupId, yearId, mountNum)

	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err := json.Unmarshal([]byte(body), &scheduleIn); err != nil {
		return nil, err
	}

	if len(scheduleIn.Sched) < 1 {
		err := fmt.Errorf("UrSEI Server error. Cant json.Unmarshal().")
		return nil, err
	}

	monthSchedule := scheduleIn.convertFormat(groupId, yearId, mountNum)

	return monthSchedule, nil
}

func (monthSchedule *MonthSchedule) saveScheduleToRedis() {
	ctx := context.Background()
	rdb := redis.NewClient(&redisConnection)

	scheduleJson, err := json.Marshal(monthSchedule)
	if err != nil {
		panic(err)
	}

	redisName := fmt.Sprintf("%d_%d_%d", monthSchedule.GroupId, monthSchedule.YearId, monthSchedule.Month)

	rdb.Set(ctx, redisName, scheduleJson, 0)
}

func getMonthScheduleFromDatabase(groupId, yearId, monthNum int) *MonthSchedule {
	ctx := context.Background()
	rdb := redis.NewClient(&redisConnection)

	redisName := fmt.Sprintf("%d_%d_%d", groupId, yearId, monthNum)

	scheduleJson, err := rdb.Get(ctx, redisName).Result()
	if err != nil {
		log.Println(err)
	}

	var monthSchedule MonthSchedule
	if err := json.Unmarshal([]byte(scheduleJson), &monthSchedule); err != nil {
		log.Println(err)
	}

	return &monthSchedule
}

func (monthSchedule *MonthSchedule) printMonthSchedule() {
	fmt.Printf("Группа: %v, Месяц: %v\n", monthSchedule.GroupId, monthSchedule.Month)
	for _, day := range monthSchedule.Days {
		fmt.Printf("Дата: %v, День: %v\n", day.Date, day.DayOfWeek)
		fmt.Printf("Расписание:\n")
		for _, pair := range day.Pairs {
			fmt.Printf(
				"Предмет: %v, Преподаватель: %v, Тип занятия: %v, Время начала: %v, Аудитория: %v\n",
				pair.Subject, pair.Teacher, pair.ClassType, pair.TimeStart, pair.Audience)
		}
		fmt.Println()
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	data := []struct {
		group int
		year  int
		month int
	}{
		{
			group: 26066,
			year:  26,
			month: 11,
		},
		{
			group: 26066,
			year:  26,
			month: 12,
		},
		{
			group: 26067,
			year:  26,
			month: 12,
		},
	}

	for _, i := range data {
		monthSchedule, err := getScheduleData(i.group, i.year, i.month)
		if err != nil {
			log.Println(err)
			continue
		}
		monthSchedule.saveScheduleToRedis()
	}

	for _, i := range data {
		schedule := getMonthScheduleFromDatabase(i.group, i.year, i.month)
		schedule.printMonthSchedule()
	}
}
