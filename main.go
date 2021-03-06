package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
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

type ScheduleInitDataIn struct {
	YearList []struct {
		YearID    int    `json:"Year_ID"`
		EduYear   string `json:"EduYear"`
		DateStart string `json:"DateStart"`
		DateEnd   string `json:"DateEnd"`
	} `json:"YearList"`
	GSTree []struct {
		YearID  int `json:"Year_ID"`
		FormEdu []struct {
			FormEduID   int    `json:"FormEdu_ID"`
			FormEduName string `json:"FormEduName"`
			Vis         bool   `json:"vis"`
			Arr         []struct {
				Curs int  `json:"Curs"`
				Vis  bool `json:"vis"`
				Arr  []struct {
					GSID      int    `json:"GS_ID"`
					CursNum   int    `json:"CursNum"`
					GSName    string `json:"GSName"`
					PlanDefID int    `json:"PlanDef_ID"`
				} `json:"arr"`
			} `json:"arr"`
		} `json:"FormEdu"`
	} `json:"GSTree"`
	Months []struct {
		MonthID     int    `json:"Month_ID"`
		MonthNumber int    `json:"MonthNumber"`
		MonthName   string `json:"MonthName"`
	} `json:"Months"`
}

type Groups []struct {
	EducationType string `json:"educationType"`
	GroupName     string `json:"groupName"`
	GroupID       int    `json:"groupID"`
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

func createRedisNameString(group, year, month int) string {
	return fmt.Sprintf("%d_%d_%d", group, year, month)
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

func (scheduleInitDataIn *ScheduleInitDataIn) convertFormat() *Groups {
	var groups Groups



	return &groups
}

func getGroupList() error {
	var scheduleInitDataIn ScheduleInitDataIn
	url := "https://www.ursei.ac.ru/Services/GetGSSchedIniData"

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err := json.Unmarshal([]byte(body), &scheduleInitDataIn); err != nil {
		return err
	}


	return nil
}

func getScheduleData(groupId, yearId, monthNum int) (*MonthSchedule, error) {
	var scheduleIn ScheduleIn
	url := fmt.Sprintf(
		"https://www.ursei.ac.ru/Services/GetGsSched?grpid=%d&yearid=%d&monthnum=%d",
		groupId, yearId, monthNum)

	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err := json.Unmarshal([]byte(body), &scheduleIn); err != nil {
		var scheduleInEmpty struct{ string }
		if err := json.Unmarshal([]byte(body), &scheduleInEmpty); err != nil {
			return nil, err
		}

		err := fmt.Errorf("Schedule is empty")
		return nil, err
	}

	if len(scheduleIn.Sched) < 1 {
		err := fmt.Errorf("UrSEI Server error. Cant json.Unmarshal().")
		return nil, err
	}

	log.Printf("Getting %d_%d_%d schedule\n", groupId, yearId, monthNum)
	monthSchedule := scheduleIn.convertFormat(groupId, yearId, monthNum)

	return monthSchedule, nil
}

func (monthSchedule *MonthSchedule) saveScheduleToRedis() error {
	ctx := context.Background()
	rdb := redis.NewClient(&redisConnection)

	scheduleJson, err := json.Marshal(monthSchedule)
	if err != nil {
		return err
	}

	redisName := createRedisNameString(monthSchedule.GroupId, monthSchedule.YearId, monthSchedule.Month)

	rdb.Set(ctx, redisName, scheduleJson, 0)
	return nil
}

func getMonthScheduleFromDatabase(groupId, yearId, monthNum int) (*MonthSchedule, error) {
	ctx := context.Background()
	rdb := redis.NewClient(&redisConnection)

	redisName := createRedisNameString(groupId, yearId, monthNum)

	scheduleJson, err := rdb.Get(ctx, redisName).Result()
	if err != nil {
		return nil, err
	}

	var monthSchedule MonthSchedule
	if err := json.Unmarshal([]byte(scheduleJson), &monthSchedule); err != nil {
		return nil, err
	}

	return &monthSchedule, nil
}

func (monthSchedule *MonthSchedule) printMonthSchedule() {
	fmt.Printf("????????????: %v, ??????????: %v\n", monthSchedule.GroupId, monthSchedule.Month)
	for _, day := range monthSchedule.Days {
		fmt.Printf("????????: %v, ????????: %v\n", day.Date, day.DayOfWeek)
		fmt.Printf("????????????????????:\n")
		for _, pair := range day.Pairs {
			fmt.Printf(
				"??????????????: %v, ??????????????????????????: %v, ?????? ??????????????: %v, ?????????? ????????????: %v, ??????????????????: %v\n",
				pair.Subject, pair.Teacher, pair.ClassType, pair.TimeStart, pair.Audience)
		}
		fmt.Println()
	}
}

func checkMonthScheduleExists(group, year, month int) (bool, error) {
	ctx := context.Background()
	rdb := redis.NewClient(&redisConnection)

	redisName := createRedisNameString(group, year, month)

	exists, err := rdb.Exists(ctx, redisName).Result()
	if err != nil {
		return false, err

	}

	if exists != 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func getPrevMonths(group, year, month int) error {
	studyYearMonths := []int{9, 10, 11, 12, 1, 2, 3, 4, 5, 6}

	isStudyYear := false
	for _, i := range studyYearMonths {
		if i == month {
			isStudyYear = true
			break
		}
	}

	if !isStudyYear {
		err := fmt.Errorf("Month is not in study year")
		return err
	}

	for i := 0; studyYearMonths[i] != month; i++ {
		currentMonth := studyYearMonths[i]
		exists, err := checkMonthScheduleExists(group, year, currentMonth)
		if err != nil {
			log.Println(err)
			continue
		}

		if exists {
			continue
		}

		monthSchedule, err := getScheduleData(group, year, currentMonth)
		if err != nil {
			log.Println(err)
			continue
		}

		if monthSchedule.saveScheduleToRedis(); err != nil {
			log.Println(err)
		}
	}

	return nil
}

func runWorkers() {
	type scheduleInfo struct {
		group int
		year  int
		month int
	}

	data := []scheduleInfo{
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

	var wg sync.WaitGroup
	for _, i := range data {
		wg.Add(1)
		go func(si scheduleInfo) {
			defer wg.Done()
			getPrevMonths(si.group, si.year, si.month)
			monthSchedule, err := getScheduleData(si.group, si.year, si.month)
			if err != nil {
				if err.Error() == "Schedule is empty" {
					return
				}

				log.Println(err)
				return
			}
			if monthSchedule.saveScheduleToRedis(); err != nil {
				log.Println(err)
			}
		}(i)
	}

	wg.Wait()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	runWorkers()
}
