package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net/http"
	_ "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var bitMap = make(map[int64]bool)
var finalUrlList []string

type analyze struct {
	Reg   string
	Start int
	End   int
}

var analyzeList = map[string]analyze{
	"Id":          {`course_id=.*?"`, 10, 1},
	"Url":         analyze{`/pc/course.html\?course_id=.*?"`, 0, 1},
	"Title":       analyze{`"course-title".*?<span>.*?</span>`, 51, 7},
	"Subject":     analyze{`"course-subject">.*?</i>`, 17, 4},
	"Time":        analyze{`"course-times">.*?</p>`, 15, 4},
	"TeacherName": analyze{`"teacher-name">.*?</p>`, 15, 4},
	"Price":       analyze{`"course-price--cost">.*?</span>`, 21, 7},
}

//使用database/sql包中的Open连接数据库
var db *sql.DB


type CourseDetail struct {
	Id          int    `json:"id"`
	Url         string `json:"url"`
	FatherTitle       string `json:"father_title"`
	Subject     string `json:"subject"`
	Time        string `json:"time"`
	TeacherName string `json:"teacher_name"`
	Price       string `json:"price"`
	ChildTitile string `json:"title"`
	BgTime     int `json:"bgtime"`
	TeacherNameDetail     string `json:"tname"`
	AddTime     int64       `json:"add_time"`
}

type Course struct {
	Id          int    `json:"id"`
	Url         string `json:"url"`
	Title       string `json:"title"`
	Subject     string `json:"subject"`
	Time        string `json:"time"`
	TeacherName string `json:"teacher_name"`
	Price       string `json:"price"`
}

func StructToJsonDemo(course Course) {
	jsonBytes, err := json.Marshal(course)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(jsonBytes))
}

func main() {
	handleUrl("https://fudao.qq.com")
	var normalUrlList []string
	var specialUrlList []string
	for _, url := range finalUrlList {
		subjectFlag := strings.Index(url, "subject")
		courseFlag := strings.Index(url, "course")
		if subjectFlag != -1 {
			normalUrlList = append(normalUrlList, url)
			continue
		}
		if courseFlag != -1 {
			specialUrlList = append(specialUrlList, url)
			continue
		}
	}

	for _,url:=range specialUrlList{
		var courseList []Course
		var courseDetailList []CourseDetail
		courseList=collectData(url)
		if len(courseList)>0{
			courseDetailList=transToSqlStruct(courseList)
			insertDb(courseDetailList)
		}
	}
	for _,url:=range normalUrlList{
		var courseList []Course
		var courseDetailList []CourseDetail
		courseList=collectData(url)
		if len(courseList)>0{
			courseDetailList=transToSqlStruct(courseList)
			insertDb(courseDetailList)
		}
	}

}

func getDbConnect() *sql.DB{
	var err error
	if db==nil{
		db, err = sql.Open("mysql", "root:wananle0@tcp(localhost:3306)/test?charset=utf8")
		if err != nil {
			fmt.Println("连接数据库失败:", err)
			os.Exit(1)
		}
	}
	return db
}

func insertDb(courseDetailList []CourseDetail) {
	fmt.Println(courseDetailList)
	db=getDbConnect()
	//使用DB结构体实例方法Prepare预处理插入,Prepare会返回一个stmt对象
	stmt,err := db.Prepare("insert into `courses`(course_id,url,father_title,subject,`time`,teacher_name,`price`,child_title,bg_time,teacher_name_detail,add_time)values(?,?,?,?,?,?,?,?,?,?,?)")
	if err!=nil{
		fmt.Println("预处理失败:",err)
		return
	}
	for i:=0;i<len(courseDetailList) ; i++ {
		//使用Stmt对象执行预处理参数
		result,err := stmt.Exec(courseDetailList[i].Id,courseDetailList[i].Url,courseDetailList[i].FatherTitle,courseDetailList[i].Subject,courseDetailList[i].Time,
			courseDetailList[i].TeacherName,courseDetailList[i].Price,
			courseDetailList[i].ChildTitile,courseDetailList[i].BgTime,courseDetailList[i].TeacherNameDetail,courseDetailList[i].AddTime)

		if err!=nil{
			fmt.Println("执行预处理失败:",err)
			return
		}else{
			rows,_ := result.RowsAffected()
			fmt.Println("执行成功,影响行数",rows,"行" )
		}
	}
}


func transToSqlStruct(courseList []Course)[]CourseDetail  {
	//把同一门课老师合并一下
	for i:=len(courseList)-1;i>=0 ;i--  {
		if courseList[i].Id==0&&courseList[i].TeacherName!=""&&courseList[i].TeacherName!="待分配"{
			if !strings.Contains(courseList[i-1].TeacherName,courseList[i].TeacherName){
				courseList[i-1].TeacherName+=","+courseList[i].TeacherName
			}
		}
		if courseList[i].Id==0{
			courseList = append(courseList[:i], courseList[i+1:]...)
		}
	}

	var resp *http.Response
	var body []byte
	var courseDetail []CourseDetail
	//解析时间吧，这个有点麻烦= =
	addtime:=time.Now().Unix()
	for i:=0;i<len(courseList);i++{
		resp, _ = http.Get(courseList[i].Url)
		body, _ = ioutil.ReadAll(resp.Body)
		str := byteString(body)
		re := regexp.MustCompile(`"directory":.*?"detail"`)
		res := re.FindAllStringSubmatch(str, -1)
		fmt.Println(res[0][0][12:len(res[0][0])-9])
		json.Unmarshal([]byte(res[0][0][12:len(res[0][0])-9]), &courseDetail)
		for j:=0;j<len(courseDetail);j++ {
			courseDetail[j].Url=courseList[i].Url
			courseDetail[j].Id=courseList[i].Id
			courseDetail[j].FatherTitle=courseList[i].Title
			courseDetail[j].Subject=courseList[i].Subject
			courseDetail[j].TeacherName=courseList[i].TeacherName
			courseDetail[j].Price=courseList[i].Price
			courseDetail[j].AddTime= addtime
			courseDetail[j].Time= courseList[i].Time
		}
	}

	defer resp.Body.Close()
	return  courseDetail
}


func handleUrl(url string) {

	var urlList []string
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	str := byteString(body)

	re := regexp.MustCompile(`<a href="//fudao.qq.com.*?"`)
	//re := regexp.MustCompile(`<a href="//.*?"`)
	matched := re.FindAllStringSubmatch(str, -1)
	urlToStore := ""
	//这是取得子网址
	for _, match := range matched {
		urlToStore = "http:" + match[0][9:][:len(match[0][9:])-1]
		//bitMap判重,布隆过滤器也是类似的，就不写那么多了
		if bitMapHandle(urlToStore) == false {
			//fmt.Println(urlToStore)
			urlList = append(urlList, urlToStore)
			handleUrl(urlToStore)
		}
	}
}

func bitMapHandle(url string) bool {
	data := []byte(url)
	has := md5.Sum(data)
	var bt int64
	bt = int64(has[0])
	for i := 11; i < 16; i++ {
		bt = int64(has[i]) + (2<<6)*bt
	}

	if bitMap[bt] == true {
		return true
	} else {
		bitMap[bt] = true
		finalUrlList = append(finalUrlList, url)
		return false
	}

}

func byteString(p []byte) string {
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}

func collectData(url string)[]Course {
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	str := byteString(body)

	var subjectCount []int
	var temp [][]string
	var courseList []Course

	for key, value := range analyzeList {
		temp = [][]string{}
		if key == "Title" {
			continue
		}
		if value.Reg != "" {
			re := regexp.MustCompile(value.Reg)
			res := re.FindAllStringSubmatch(str, -1)
			temp = res[0:]
		}

		if len(temp) > len(courseList) {
			courses := make([]Course, len(temp))
			courseList = append(courseList, courses...)
		}
		for i := 0; i < len(temp); i++ {
			temp[i][0] = temp[i][0][value.Start : len(temp[i][0])-value.End]

			// TODO 这一块可以考虑反射来做
			switch key {
			case "Id":
				courseList[i].Id, _ = strconv.Atoi(temp[i][0])
			case "Url":
				courseList[i].Url = "http://fudao.qq.com"+temp[i][0]
			case "Subject":
				courseList[i].Subject = temp[i][0]
			case "Time":
				courseList[i].Time = temp[i][0]
			case "TeacherName":
				courseList[i].TeacherName = temp[i][0]
			case "Price":
				courseList[i].Price = temp[i][0]
			}

			if key == "Subject" {
				subjectCount = append(subjectCount, len(temp[i][0]))
			}
		}
	}
	temp = [][]string{}
	if analyzeList["Title"].Reg != "" {
		re := regexp.MustCompile(analyzeList["Title"].Reg)
		res := re.FindAllStringSubmatch(str, -1)
		temp = res[0:]
	}
	for i := 0; i < len(temp); i++ {
		temp[i][0] = temp[i][0][analyzeList["Title"].Start+subjectCount[i] : len(temp[i][0])-analyzeList["Title"].End]
		courseList[i].Title = temp[i][0]
	}
	return courseList
}
