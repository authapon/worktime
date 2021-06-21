package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	mw "github.com/authapon/mwebserv"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type dataChkIn struct {
	Uid       int64
	Epassport string
	Fname     string
	Lname     string
	Timex     string
	Worktime  string
	Day       string
	Ip        string
	Groupid   string
	Groupname string
	Pic       string
	Late      int
	Absent    int
}
type groupData struct {
	Groupid   string
	Groupname string
}

var (
	weekdays []string = []string{"อาทิตย์", "จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์", "เสาร์"}
	months   []string = []string{"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม"}
)

func getHMS(d string) (int, int, int) {
	tx := strings.Split(d, " ")
	dat := strings.Split(tx[1], ":")
	hh, _ := strconv.Atoi(dat[0])
	mm, _ := strconv.Atoi(dat[1])
	ss, _ := strconv.Atoi(dat[2])

	return hh, mm, ss
}
func getWorkingDayRange(day int, month int, year int, day2 int, month2 int, year2 int) ([]int, error) {
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []int{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	time1 := fmt.Sprintf("%d-%d-%d 00:00:00", year, month, day)
	time2 := fmt.Sprintf("%d-%d-%d 23:59:59", year2, month2, day2)

	st, _ := con.Prepare("SELECT distinct day(`timex`) as dayx, month(`timex`) as monthx, year(`timex`) as yearx from `worktime` where `timex`>=? and `timex`<=? order by timex;")
	rows, err := st.Query(time1, time2)
	if err != nil {
		return []int{}, errors.New("Error to Database")
	}
	defer rows.Close()

	data := []int{}

	for rows.Next() {
		dayx := 0
		monthx := 0
		yearx := 0
		rows.Scan(&dayx, &monthx, &yearx)
		data = append(data, (yearx*10000)+(monthx*100)+dayx)
	}

	return data, nil
}

func getWorkingDay(month int, year int) ([]int, error) {
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []int{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	st, _ := con.Prepare("SELECT distinct day(`timex`) as dayx from `worktime` where month(`timex`)=? and year(`timex`)=? order by timex;")
	rows, err := st.Query(month, year)
	if err != nil {
		return []int{}, errors.New("Error to Database")
	}
	defer rows.Close()

	data := []int{}

	for rows.Next() {
		day := 0
		rows.Scan(&day)
		data = append(data, day)
	}

	return data, nil
}

func getLate(timex string) int {
	h, m, _ := getHMS(timex)
	dtime := (h * 60) + m
	m2 := lateConf % 100
	ftime := (((lateConf - (m2)) / 100) * 60) + m2

	late := dtime - ftime
	if late < 0 {
		return 0
	}

	return late
}

func getGroups() ([]groupData, error) {
	groups := []groupData{}
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []groupData{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	st, _ := con.Prepare("select `groupid`, `groupname` from `groups`;")
	rows, err := st.Query()
	if err != nil {
		return []groupData{}, errors.New("Error to Database")
	}
	defer rows.Close()

	for rows.Next() {
		var gdata groupData
		rows.Scan(&gdata.Groupid, &gdata.Groupname)
		groups = append(groups, gdata)
	}

	return groups, nil
}

func getGroupname(groupid string) (string, error) {
	gname := ""
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return "", errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	st, _ := con.Prepare("select `groupname` from `groups` where `groupid`=?;")
	if err := st.QueryRow(groupid).Scan(&gname); err != nil {
		return "", errors.New("Error to Database")
	}

	return gname, nil
}

func addUserActive(userx []dataChkIn, gid string) []dataChkIn {
	q := ""
	if len(userx) == 0 {
		q = "select `epassport`, `name`, `surname` from `users` where `active`=1 and `groupid`=?;"
	} else {
		co := 0
		u := ""
		for i := range userx {
			if co == 0 {
				u = fmt.Sprintf("'%s'", userx[i].Epassport)
				co = 1
			} else {
				u = u + fmt.Sprintf(",'%s'", userx[i].Epassport)
			}
		}
		q = fmt.Sprintf("select `epassport`, `name`, `surname` from `users` where `active`=1 and `groupid`=? and `epassport` not in (%s);", u)
	}
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return userx
	}
	defer con.Close()

	st, _ := con.Prepare(q)
	rows, err := st.Query(gid)
	if err != nil {
		return userx
	}
	defer rows.Close()

	for rows.Next() {
		var d dataChkIn
		rows.Scan(&d.Epassport, &d.Fname, &d.Lname)
		userx = append(userx, d)
	}

	return userx
}
func getUsersWorkingRange(groupid string, day int, month int, year int, day2 int, month2 int, year2 int) ([]dataChkIn, error) {
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []dataChkIn{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	time1 := fmt.Sprintf("%d-%d-%d 00:00:00", year, month, day)
	time2 := fmt.Sprintf("%d-%d-%d 23:59:59", year2, month2, day2)

	st, _ := con.Prepare("select distinct `worktime`.`epassport` as epassport, `users`.`name` as fname, `users`.`surname` as lname from `worktime` left join (`users`) on (`worktime`.`epassport`=`users`.`epassport`) where `worktime`.`timex`>=? and `worktime`.`timex`<=? and `users`.`groupid`=?;")
	rows, err := st.Query(time1, time2, groupid)
	if err != nil {
		return []dataChkIn{}, errors.New("Error to Database")
	}
	defer rows.Close()

	data := []dataChkIn{}

	for rows.Next() {
		var d dataChkIn
		rows.Scan(&d.Epassport, &d.Fname, &d.Lname)
		data = append(data, d)
	}

	return data, nil
}

func getUsersWorking(groupid string, month int, year int) ([]dataChkIn, error) {
	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []dataChkIn{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	st, _ := con.Prepare("select distinct `worktime`.`epassport` as epassport, `users`.`name` as fname, `users`.`surname` as lname from `worktime` left join (`users`) on (`worktime`.`epassport`=`users`.`epassport`) where month(`worktime`.`timex`)=? and year(`worktime`.`timex`)=? and `users`.`groupid`=?;")
	rows, err := st.Query(month, year, groupid)
	if err != nil {
		return []dataChkIn{}, errors.New("Error to Database")
	}
	defer rows.Close()

	data := []dataChkIn{}

	for rows.Next() {
		var d dataChkIn
		rows.Scan(&d.Epassport, &d.Fname, &d.Lname)
		data = append(data, d)
	}

	return data, nil
}

func getCheckinDate(datetxt string) ([]dataChkIn, error) {
	dd1 := datetxt + " 00:00:00"
	dd2 := datetxt + " 23:59:59"

	data := []dataChkIn{}
	epassAll := []string{}

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []dataChkIn{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	st1, _ := con.Prepare("select `uid`,`worktime`.`epassport` as `epassport`,`name`,`surname`,`timex`,`ip`,`users`.`groupid` as `groupid`,`groupname`,`pic` from `worktime` left join (users, groups) on (worktime.epassport=users.epassport and users.groupid=groups.groupid) where `timex` >= ? and `timex` <= ? order by `timex`;")
	rows1, err := st1.Query(dd1, dd2)
	if err != nil {
		return []dataChkIn{}, errors.New("Error to Database")
	}
	defer rows1.Close()

	havesomeone := false
	for rows1.Next() {
		havesomeone = true
		var d dataChkIn
		var pic string
		rows1.Scan(&d.Uid, &d.Epassport, &d.Fname, &d.Lname, &d.Timex, &d.Ip, &d.Groupid, &d.Groupname, &pic)
		picData := strings.Split(pic, ":")
		d.Pic = picData[1]
		d.Late = getLate(d.Timex)
		h, m, _ := getHMS(d.Timex)
		d.Worktime = fmt.Sprintf("%02d:%02d", h, m)
		epassAll = append(epassAll, d.Epassport)
		data = append(data, d)
	}

	epassTxt := ""
	for i := range epassAll {
		if epassTxt == "" {
			epassTxt = epassTxt + fmt.Sprintf("'%s'", epassAll[i])
		} else {
			epassTxt = epassTxt + "," + fmt.Sprintf("'%s'", epassAll[i])
		}
	}

	q := ""
	if havesomeone {
		q = "select `epassport`, `name`, `surname`, `groupid` from `users` where `epassport` not in (" + epassTxt + ") and `active`=1;"
	} else {
		q = "select `epassport`, `name`, `surname`, `groupid` from `users` where `active`=1;"
	}
	st2, _ := con.Prepare(q)
	rows2, err := st2.Query()
	if err != nil {
		return []dataChkIn{}, errors.New("Error to Database")
	}
	defer rows2.Close()

	for rows2.Next() {
		var d dataChkIn
		rows2.Scan(&d.Epassport, &d.Fname, &d.Lname, &d.Groupid)
		d.Worktime = "-----"
		d.Late = 0
		d.Ip = "-----"
		data = append(data, d)
	}

	return data, nil
}

func extractDateTimeFromText(datetxt string, flag bool) (int, int, int, time.Time, error) {
	dtxtSplit := strings.Split(datetxt, "/")
	if len(dtxtSplit) != 3 {
		return 0, 0, 0, time.Now(), errors.New("Wrong data !!!")
	}

	day, err := strconv.Atoi(dtxtSplit[0])
	if err != nil {
		return 0, 0, 0, time.Now(), errors.New("Wrong data !!!")
	}

	month, err := strconv.Atoi(dtxtSplit[1])
	if err != nil {
		return 0, 0, 0, time.Now(), errors.New("Wrong data !!!")
	}

	year, err := strconv.Atoi(dtxtSplit[2])
	if err != nil {
		return 0, 0, 0, time.Now(), errors.New("Wrong data !!!")
	}
	year = year - 543

	tt := ""
	if flag {
		tt = "23:59:59"
	} else {
		tt = "00:00:00"
	}

	t, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d %s", year, month, day, tt))
	if err != nil {
		return 0, 0, 0, time.Now(), errors.New("Wrong data !!!")
	}

	return day, month, year, t, nil
}

func getPersonWorktimeRange(epassport string, day int, month int, year int, day2 int, month2 int, year2 int) ([]dataChkIn, error) {
	data := []dataChkIn{}

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []dataChkIn{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	time1 := fmt.Sprintf("%d-%d-%d 00:00:00", year, month, day)
	time2 := fmt.Sprintf("%d-%d-%d 23:59:59", year2, month2, day2)

	st, _ := con.Prepare("select `uid`,`worktime`.`epassport` as `epassport`,`name`,`surname`,`timex`,`ip`,`users`.`groupid` as `groupid`,`groupname`,`pic` from `worktime` left join (users, groups) on (worktime.epassport=users.epassport and users.groupid=groups.groupid) where `users`.`epassport`=? and `timex`>=? and `timex`<=? order by `timex`;")
	rows, err := st.Query(epassport, time1, time2)
	if err != nil {
		return []dataChkIn{}, errors.New("Error to Database")
	}
	defer rows.Close()

	workingDay, err := getWorkingDayRange(day, month, year, day2, month2, year2)
	if err != nil {
		return []dataChkIn{}, errors.New("Internal Error to get Working days")
	}

	cday := 0
	var dumpData dataChkIn

	for rows.Next() {
		var d dataChkIn
		var pic string
		rows.Scan(&d.Uid, &d.Epassport, &d.Fname, &d.Lname, &d.Timex, &d.Ip, &d.Groupid, &d.Groupname, &pic)
		picData := strings.Split(pic, ":")
		d.Pic = picData[1]
		d.Late = getLate(d.Timex)
		h, m, _ := getHMS(d.Timex)
		d.Worktime = fmt.Sprintf("%02d:%02d", h, m)
		t, err := time.Parse("2006-01-02 15:04:05", d.Timex)
		if err != nil {
			return []dataChkIn{}, errors.New("Internal Error")
		}
		weekday := int(t.Weekday())
		weekdayTxt := weekdays[weekday]
		d.Day = fmt.Sprintf("%s ที่ %d %s %d", weekdayTxt, t.Day(), months[int(t.Month())-1], t.Year()+543)

		for {
			if cday >= len(workingDay) {
				break
			}
			if (t.Year()*10000)+(int(t.Month())*100)+t.Day() == workingDay[cday] {
				cday = cday + 1
				break

			} else {
				var dd dataChkIn
				dd.Uid = d.Uid
				dd.Epassport = d.Epassport
				dd.Fname = d.Fname
				dd.Lname = d.Lname
				dd.Timex = "-----"
				dd.Ip = "-----"
				dd.Groupid = d.Groupid
				dd.Groupname = d.Groupname
				dd.Pic = ""
				dd.Worktime = "-----"
				dd.Late = 0

				dayx := workingDay[cday] % 100
				monthx := (workingDay[cday] / 100) % 100
				yearx := workingDay[cday] / 10000

				tt, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d 00:00:00", yearx, monthx, dayx))
				if err != nil {
					return []dataChkIn{}, errors.New("Internal Error")
				}
				weekday := int(tt.Weekday())
				weekdayTxt := weekdays[weekday]
				dd.Day = fmt.Sprintf("%s ที่ %d %s %d", weekdayTxt, tt.Day(), months[int(tt.Month())-1], tt.Year()+543)

				data = append(data, dd)
				cday = cday + 1
			}
		}

		dumpData.Uid = d.Uid
		dumpData.Epassport = d.Epassport
		dumpData.Fname = d.Fname
		dumpData.Lname = d.Lname
		dumpData.Groupid = d.Groupid
		dumpData.Groupname = d.Groupname

		data = append(data, d)
	}

	if cday < len(workingDay) {
		for {
			if cday >= len(workingDay) {
				break
			}
			var dd dataChkIn
			dd.Uid = dumpData.Uid
			dd.Epassport = dumpData.Epassport
			dd.Fname = dumpData.Fname
			dd.Lname = dumpData.Lname
			dd.Timex = "-----"
			dd.Ip = "-----"
			dd.Groupid = dumpData.Groupid
			dd.Groupname = dumpData.Groupname
			dd.Pic = ""
			dd.Worktime = "-----"
			dd.Late = 0

			dayx := workingDay[cday] % 100
			monthx := (workingDay[cday] / 100) % 100
			yearx := workingDay[cday] / 10000

			tt, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d 00:00:00", yearx, monthx, dayx))
			if err != nil {
				return []dataChkIn{}, errors.New("Internal Error")
			}
			weekday := int(tt.Weekday())
			weekdayTxt := weekdays[weekday]
			dd.Day = fmt.Sprintf("%s ที่ %d %s %d", weekdayTxt, tt.Day(), months[int(tt.Month())-1], tt.Year()+543)

			data = append(data, dd)
			cday = cday + 1
		}
	}

	return data, nil
}

func getPersonWorktime(epassport string, month int, year int) ([]dataChkIn, error) {
	data := []dataChkIn{}

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		return []dataChkIn{}, errors.New("Cannot Connect to Database")
	}
	defer con.Close()

	st, _ := con.Prepare("select `uid`,`worktime`.`epassport` as `epassport`,`name`,`surname`,`timex`,`ip`,`users`.`groupid` as `groupid`,`groupname`,`pic` from `worktime` left join (users, groups) on (worktime.epassport=users.epassport and users.groupid=groups.groupid) where `users`.`epassport`=? and month(`timex`)=? and year(`timex`)=? order by `timex`;")
	rows, err := st.Query(epassport, month, year)
	if err != nil {
		return []dataChkIn{}, errors.New("Error to Database")
	}
	defer rows.Close()

	workingDay, err := getWorkingDay(month, year)
	if err != nil {
		return []dataChkIn{}, errors.New("Internal Error to get Working days")
	}

	cday := 0
	var dumpData dataChkIn

	for rows.Next() {
		var d dataChkIn
		var pic string
		rows.Scan(&d.Uid, &d.Epassport, &d.Fname, &d.Lname, &d.Timex, &d.Ip, &d.Groupid, &d.Groupname, &pic)
		picData := strings.Split(pic, ":")
		d.Pic = picData[1]
		d.Late = getLate(d.Timex)
		h, m, _ := getHMS(d.Timex)
		d.Worktime = fmt.Sprintf("%02d:%02d", h, m)
		t, err := time.Parse("2006-01-02 15:04:05", d.Timex)
		if err != nil {
			return []dataChkIn{}, errors.New("Internal Error")
		}
		weekday := int(t.Weekday())
		weekdayTxt := weekdays[weekday]
		d.Day = fmt.Sprintf("%s ที่ %d", weekdayTxt, t.Day())

		for {
			if cday >= len(workingDay) {
				break
			}
			if t.Day() == workingDay[cday] {
				cday = cday + 1
				break

			} else {
				var dd dataChkIn
				dd.Uid = d.Uid
				dd.Epassport = d.Epassport
				dd.Fname = d.Fname
				dd.Lname = d.Lname
				dd.Timex = "-----"
				dd.Ip = "-----"
				dd.Groupid = d.Groupid
				dd.Groupname = d.Groupname
				dd.Pic = ""
				dd.Worktime = "-----"
				dd.Late = 0

				tt, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d 00:00:00", year, month, workingDay[cday]))
				if err != nil {
					return []dataChkIn{}, errors.New("Internal Error")
				}
				weekday := int(tt.Weekday())
				weekdayTxt := weekdays[weekday]
				dd.Day = fmt.Sprintf("%s ที่ %d", weekdayTxt, tt.Day())

				data = append(data, dd)
				cday = cday + 1
			}
		}

		dumpData.Uid = d.Uid
		dumpData.Epassport = d.Epassport
		dumpData.Fname = d.Fname
		dumpData.Lname = d.Lname
		dumpData.Groupid = d.Groupid
		dumpData.Groupname = d.Groupname

		data = append(data, d)
	}

	if cday < len(workingDay) {
		for {
			if cday >= len(workingDay) {
				break
			}
			var dd dataChkIn
			dd.Uid = dumpData.Uid
			dd.Epassport = dumpData.Epassport
			dd.Fname = dumpData.Fname
			dd.Lname = dumpData.Lname
			dd.Timex = "-----"
			dd.Ip = "-----"
			dd.Groupid = dumpData.Groupid
			dd.Groupname = dumpData.Groupname
			dd.Pic = ""
			dd.Worktime = "-----"
			dd.Late = 0

			tt, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d 00:00:00", year, month, workingDay[cday]))
			if err != nil {
				return []dataChkIn{}, errors.New("Internal Error")
			}
			weekday := int(tt.Weekday())
			weekdayTxt := weekdays[weekday]
			dd.Day = fmt.Sprintf("%s ที่ %d", weekdayTxt, tt.Day())

			data = append(data, dd)
			cday = cday + 1
		}
	}

	return data, nil
}

func indexPage(c *mw.MContext) {
	c.Render("index.html", make(map[string]interface{}))
}

func checkinPage(c *mw.MContext) {
	c.R.ParseForm()
	uid := int64(0)
	username := strings.ToLower(strings.TrimSpace(c.R.Form.Get("username")))
	fullname := c.R.Form.Get("fullname")
	pic := c.R.Form.Get("pic")
	fmt.Printf("%s - %s\n", username, fullname)
	if username == "" {
		c.Redirect("/")
	}

	t := time.Now()
	day := strconv.FormatInt(int64(t.Day()), 10)
	month := int(t.Month())
	monthnum := strconv.FormatInt(int64(month), 10)
	year := strconv.FormatInt(int64(t.Year()), 10)
	hour := fmt.Sprintf("%02d", t.Hour())
	minute := fmt.Sprintf("%02d", t.Minute())
	second := fmt.Sprintf("%02d", t.Second())

	datetxt := year + "-" + monthnum + "-" + day
	dd1 := datetxt + " 00:00:00"
	dd2 := datetxt + " 23:59:59"

	workin := datetxt + " " + hour + ":" + minute

	ip := c.RemoteAddr()

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		c.WriteString("Cannot Connect to Database")
		return
	}
	defer con.Close()

	st1, _ := con.Prepare("select `uid` from `worktime` where `epassport`=? and `timex` >= ? and `timex` <= ?;")
	rows1, err := st1.Query(username, dd1, dd2)
	if err != nil {
		c.WriteString("Error to Database")
		return
	}
	defer rows1.Close()

	for rows1.Next() {
		rows1.Scan(&uid)

		if uid > 0 {
			st2, _ := con.Prepare("delete from `worktime` where `uid`=?")
			_, _ = st2.Exec(uid)
		}
	}

	st3, _ := con.Prepare("insert into `worktime` (`epassport`, `timex`, `ip`, `pic`) values (?,?,?,?);")
	_, _ = st3.Exec(username, workin, ip, pic)

	late := ((t.Hour() * 60) + t.Minute()) - (((lateConf - (lateConf % 100)) / 100 * 60) + (lateConf % 100))
	if late < 0 {
		late = 0
	}

	dataRender := make(map[string]interface{})
	dataRender["username"] = username
	dataRender["fullname"] = fullname
	dataRender["pic"] = pic
	dataRender["ip"] = ip
	dataRender["hour"] = hour
	dataRender["minute"] = minute
	dataRender["second"] = second
	dataRender["late"] = late

	c.Render("checkin.html", dataRender)
}

func todayPage(c *mw.MContext) {
	t := time.Now()
	day := fmt.Sprintf("%d", t.Day())
	month := fmt.Sprintf("%d", int(t.Month()))
	year := fmt.Sprintf("%d", t.Year())
	year2 := fmt.Sprintf("%d", t.Year()+543)

	datetxt := year + "-" + month + "-" + day
	dataChkInToday, err := getCheckinDate(datetxt)
	if err != nil {
		c.WriteString(fmt.Sprintf("%s", err))
		return
	}

	groups, err := getGroups()
	if err != nil {
		c.WriteString(fmt.Sprintf("%s", err))
		return
	}

	dTab := ""
	if len(groups) > 1 {
		dTab = groups[0].Groupid
	}

	dataRender := make(map[string]interface{})
	dataRender["month"] = month
	dataRender["year"] = year2
	dataRender["groups"] = groups
	dataRender["dataChkIn"] = dataChkInToday
	dataRender["dTab"] = dTab

	c.Render("todayPage.html", dataRender)
}

func reportPage(c *mw.MContext) {
	t := time.Now()
	day := fmt.Sprintf("%d", t.Day())
	month := int(t.Month())
	monthnum := fmt.Sprintf("%d", month)
	year := fmt.Sprintf("%d", t.Year()+543)

	today := day + "/" + monthnum + "/" + year

	groups, err := getGroups()
	if err != nil {
		c.WriteString(fmt.Sprintf("%s", err))
	}

	dataRender := make(map[string]interface{})
	dataRender["today"] = today
	dataRender["groups"] = groups

	c.Render("reportPage.html", dataRender)
}

func genReportPage(c *mw.MContext) {
	c.R.ParseForm()
	special := strings.TrimSpace(c.R.Form.Get("special"))
	dtxt := strings.TrimSpace(c.R.Form.Get("datepicker"))
	dtxt2 := strings.TrimSpace(c.R.Form.Get("datepicker2"))
	groupid := c.R.Form.Get("groupid")
	gname, err := getGroupname(groupid)
	if err != nil {
		c.WriteString(fmt.Sprintf("%s", err))
		return
	}

	dtxtSplit := strings.Split(dtxt, "/")
	if len(dtxtSplit) == 3 && dtxt2 == "" {

		day, month, year, t, err := extractDateTimeFromText(dtxt, false)
		if err != nil {
			c.WriteString("Wrong data !!!")
			return
		}

		weekdayTxt := weekdays[int(t.Weekday())]
		monthTxt := months[month-1]

		datetxt := fmt.Sprintf("(วัน %s ที่ %02d %s %d)", weekdayTxt, day, monthTxt, year+543)

		datetxtData := fmt.Sprintf("%d-%d-%d", year, month, day)
		dataChkInDate, err := getCheckinDate(datetxtData)
		if err != nil {
			c.WriteString(fmt.Sprintf("%s", err))
			return
		}

		if special == "on" {
			for i := range dataChkInDate {
				if dataChkInDate[i].Late > 0 {
					dataChkInDate[i].Late = 0
					m := lateConf % 100
					h := lateConf / 100
					dataChkInDate[i].Worktime = fmt.Sprintf("%02d:%02d", h, m)
				}
			}
		}

		dataRender := make(map[string]interface{})
		dataRender["datetxt"] = datetxt
		dataRender["month"] = fmt.Sprintf("%d", month)
		dataRender["year"] = fmt.Sprintf("%d", year+543)
		dataRender["gid"] = groupid
		dataRender["gname"] = gname
		dataRender["data"] = dataChkInDate

		c.Render("dailyreport.html", dataRender)

	} else if len(dtxtSplit) == 3 && dtxt2 != "" {

		day, month, year, t, err := extractDateTimeFromText(dtxt, false)
		if err != nil {
			c.WriteString("Wrong data !!!")
			return
		}

		day2, month2, year2, t2, err := extractDateTimeFromText(dtxt2, true)
		if err != nil {
			c.WriteString("Wrong data !!!")
			return
		}

		daytest := (year * 10000) + (month * 100) + day
		daytest2 := (year2 * 10000) + (month2 * 100) + day2
		if daytest > daytest2 {
			c.WriteString("Wrong data !!!")
			return
		}

		weekdayTxt := weekdays[int(t.Weekday())]
		monthTxt := months[month-1]

		weekday2Txt := weekdays[int(t2.Weekday())]
		month2Txt := months[month2-1]

		datetxt := fmt.Sprintf("(วัน %s ที่ %02d %s %d ถึง วัน %s ที่ %02d %s %d)", weekdayTxt, day, monthTxt, year+543, weekday2Txt, day2, month2Txt, year2+543)

		usersx, err := getUsersWorkingRange(groupid, day, month, year, day2, month2, year2)
		if err != nil {
			c.WriteString("Internal Error")
			return
		}

		users := addUserActive(usersx, groupid)

		dataSum := []dataChkIn{}
		for i := range users {
			dataPersonWorktime, err := getPersonWorktimeRange(users[i].Epassport, day, month, year, day2, month2, year2)
			if err != nil {
				c.WriteString("Error to get data")
				return
			}
			var dataSumUsers dataChkIn
			dataSumUsers.Epassport = users[i].Epassport
			dataSumUsers.Fname = users[i].Fname
			dataSumUsers.Lname = users[i].Lname
			dataSumUsers.Late = 0
			dataSumUsers.Absent = 0
			for i2 := range dataPersonWorktime {
				if dataPersonWorktime[i2].Timex != "-----" {
					dataSumUsers.Late = dataSumUsers.Late + getLate(dataPersonWorktime[i2].Timex)
				} else {
					dataSumUsers.Absent = dataSumUsers.Absent + 1
				}
			}

			dataSum = append(dataSum, dataSumUsers)
		}

		dataRender := make(map[string]interface{})
		dataRender["monthtxt"] = datetxt
		dataRender["gname"] = gname
		dataRender["rpath"] = "personReport2"
		dataRender["path"] = fmt.Sprintf("%d/%d/%d/%d/%d/%d", day, month, year+543, day2, month2, year2+543)
		dataRender["data"] = dataSum

		c.Render("sumReport.html", dataRender)

	} else if len(dtxtSplit) == 2 {
		month, err := strconv.Atoi(dtxtSplit[0])
		if err != nil {
			c.WriteString("Wrong data !!!")
			return
		}

		year, err := strconv.Atoi(dtxtSplit[1])
		if err != nil {
			c.WriteString("Wrong data !!!")
			return
		}
		year = year - 543

		usersx, err := getUsersWorking(groupid, month, year)
		if err != nil {
			c.WriteString("Internal Error")
			return
		}

		users := addUserActive(usersx, groupid)

		dataSum := []dataChkIn{}
		for i := range users {
			dataPersonWorktime, err := getPersonWorktime(users[i].Epassport, month, year)
			if err != nil {
				c.WriteString("Error to get data")
				return
			}
			var dataSumUsers dataChkIn
			dataSumUsers.Epassport = users[i].Epassport
			dataSumUsers.Fname = users[i].Fname
			dataSumUsers.Lname = users[i].Lname
			dataSumUsers.Late = 0
			dataSumUsers.Absent = 0
			for i2 := range dataPersonWorktime {
				if dataPersonWorktime[i2].Timex != "-----" {
					dataSumUsers.Late = dataSumUsers.Late + getLate(dataPersonWorktime[i2].Timex)
				} else {
					dataSumUsers.Absent = dataSumUsers.Absent + 1
				}
			}

			dataSum = append(dataSum, dataSumUsers)
		}

		dataRender := make(map[string]interface{})
		dataRender["monthtxt"] = months[month-1] + fmt.Sprintf(" %d", year+543)
		dataRender["gname"] = gname
		dataRender["rpath"] = "personReport"
		dataRender["path"] = fmt.Sprintf("%d/%d", month, year+543)
		dataRender["data"] = dataSum

		c.Render("sumReport.html", dataRender)

	} else {
		c.WriteString("Wrong data !!!")
	}
}

func personReportPage(c *mw.MContext) {
	epassport := c.V["epassport"]
	month := c.V["month"]
	year := c.V["year"]
	fname := ""
	lname := ""
	gid := ""
	gname := ""

	monthnum, err := strconv.Atoi(month)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	yearnum, err := strconv.Atoi(year)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	monthtxt := months[monthnum-1]

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		c.WriteString("Cannot Connect to Database")
		return
	}
	defer con.Close()

	st, _ := con.Prepare("select `users`.`name`, `users`.`surname`, `users`.`groupid`, `groups`.`groupname` from `users` left join (groups) on (`users`.`groupid`=`groups`.`groupid`) where `epassport`=?;")
	if err := st.QueryRow(epassport).Scan(&fname, &lname, &gid, &gname); err != nil {
		c.WriteString("Error to Database")
		return
	}

	dataPersonWorktime, err := getPersonWorktime(epassport, monthnum, yearnum-543)
	if err != nil {
		c.WriteString("Error to get data")
		return
	}

	dataRender := make(map[string]interface{})
	dataRender["monthtxt"] = monthtxt + fmt.Sprintf(" %d", yearnum)
	dataRender["gid"] = gid
	dataRender["gname"] = gname
	dataRender["fname"] = fname
	dataRender["lname"] = lname
	dataRender["epassport"] = epassport
	dataRender["data"] = dataPersonWorktime

	c.Render("personReport.html", dataRender)
}

func personReportRangePage(c *mw.MContext) {
	epassport := c.V["epassport"]
	day := c.V["day1"]
	month := c.V["month1"]
	year := c.V["year1"]
	day2 := c.V["day2"]
	month2 := c.V["month2"]
	year2 := c.V["year2"]

	fname := ""
	lname := ""
	gid := ""
	gname := ""

	daynum, err := strconv.Atoi(day)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	day2num, err := strconv.Atoi(day2)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	monthnum, err := strconv.Atoi(month)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	month2num, err := strconv.Atoi(month2)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	yearnum, err := strconv.Atoi(year)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	year2num, err := strconv.Atoi(year2)
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	t, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d 00:00:00", yearnum-543, monthnum, daynum))
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	t2, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%d-%02d-%02d 00:00:00", year2num-543, month2num, day2num))
	if err != nil {
		c.WriteString("Wrong data !!!")
		return
	}

	weekdayTxt := weekdays[int(t.Weekday())]
	monthTxt := months[monthnum-1]

	weekday2Txt := weekdays[int(t2.Weekday())]
	month2Txt := months[month2num-1]

	datetxt := fmt.Sprintf("(วัน %s ที่ %02d %s %s ถึง วัน %s ที่ %02d %s %s)", weekdayTxt, daynum, monthTxt, year, weekday2Txt, day2num, month2Txt, year2)

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		c.WriteString("Cannot Connect to Database")
		return
	}
	defer con.Close()

	st, _ := con.Prepare("select `users`.`name`, `users`.`surname`, `users`.`groupid`, `groups`.`groupname` from `users` left join (groups) on (`users`.`groupid`=`groups`.`groupid`) where `epassport`=?;")
	if err := st.QueryRow(epassport).Scan(&fname, &lname, &gid, &gname); err != nil {
		c.WriteString("Error to Database")
		return
	}

	dataPersonWorktime, err := getPersonWorktimeRange(epassport, daynum, monthnum, yearnum-543, day2num, month2num, year2num-543)
	if err != nil {
		c.WriteString("Error to get data")
		return
	}

	dataRender := make(map[string]interface{})
	dataRender["monthtxt"] = datetxt
	dataRender["gid"] = gid
	dataRender["gname"] = gname
	dataRender["fname"] = fname
	dataRender["lname"] = lname
	dataRender["epassport"] = epassport
	dataRender["data"] = dataPersonWorktime

	c.Render("personReport.html", dataRender)
}

func getTimeService(c *mw.MContext) {
	t := time.Now()
	weekday := int(t.Weekday())
	weekdayTxt := weekdays[weekday]
	day := fmt.Sprintf("%d", t.Day())
	month := int(t.Month())
	monthTxt := months[month-1]
	year := fmt.Sprintf("%d", t.Year()+543)
	hour := fmt.Sprintf("%02d", t.Hour())
	minute := fmt.Sprintf("%02d", t.Minute())
	second := fmt.Sprintf("%02d", t.Second())

	c.WriteString("วัน " + weekdayTxt + " ที่ " + day + " " + monthTxt + " " + year + " เวลา " + hour + " : " + minute + " : " + second)
}

func epassportService(c *mw.MContext) {
	c.R.ParseForm()
	username := c.R.Form.Get("username")
	password := c.R.Form.Get("password")
	uname := ""
	active := true

	con, err := sql.Open("mysql", mysqlConf)
	if err != nil {
		c.WriteString("Cannot Connect to Database")
		return
	}
	defer con.Close()

	st, _ := con.Prepare("select `epassport`, `active` from `users` where `epassport`=?;")
	if err := st.QueryRow(username).Scan(&uname, &active); err != nil {
		c.WriteString("none")
		return
	}

	if !active {
		c.WriteString("none")
		return
	}

	for i := 0; i < 3; i++ {
		resp, err := http.PostForm(eloginConf, url.Values{"username": {username}, "password": {password}})
		if err != nil {
			continue
		}
		p := make([]byte, resp.ContentLength)
		resp.Body.Read(p)
		rdat := make(map[string]interface{})
		if err := json.Unmarshal(p, &rdat); err != nil {
			continue
		}

		if rdat["success"].(string) != "true" {
			c.WriteString("none")
			return
		}
		c.WriteString(rdat["fullname"].(string))
		return
	}

	c.WriteString("error")
}

func canChkInService(c *mw.MContext) {
	t := time.Now()
	tnow := (t.Hour() * 100) + t.Minute()

	if (startConf <= tnow) && (tnow < stopConf) {
		c.WriteString("ok")
		return
	}

	c.WriteString("ขณะนี้หมดเวลาเข้างานแล้วครับ ...")
}
