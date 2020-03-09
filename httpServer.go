package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/cihub/seelog.v2"
	"log"
	"net/http"
	"os"
	"sort"
	_ "sync"
	"time"
)

var (
	engine = gin.Default()
)

type State int

const (
	StateOK                  State = iota
	StateContentTypeMismatch State = 1000 + iota
	StateSignatureMismatch
	StateIdentifierMismatch
)

type StateMap map[State]string

var db1 *sql.DB
var StateMapping = StateMap{
	StateOK:                  "Success",
	StateContentTypeMismatch: "Content mismatch",
	StateSignatureMismatch:   "Signature mismatch",
	StateIdentifierMismatch:  "Identifier mismatch",
}

var ContentTypeErr = errors.New("ContentTypeErr is not json object")

type Data struct {
	Identifier string `json:"identifier" binding:"required"`
	Signature  string `json:"signature"  binding:"required"`
	DataStr    string `json:"data_str"`
}

var courseList []CourseDetail1
var CourseMap = make(map[int]map[string][]CourseDetail1)

type CourseDetail1 struct {
	CourseId          int    `json:"course_id"`
	Url               string `json:"url"`
	FatherTitle       string `json:"father_title"`
	Subject           string `json:"subject"`
	Time              string `json:"time"`
	TeacherName       string `json:"teacher_name"`
	Price             string `json:"price"`
	ChildTitile       string `json:"title"`
	BgTime            int    `json:"bgtime"`
	TeacherNameDetail string `json:"tname"`
	AddTime           int64  `json:"add_time"`
}

func srvHandlerPost(c *gin.Context) {
	var (
		dat  Data
		err  error
		code State = StateOK
	)
	// handler 结束处理后返回响应体给 client 端
	defer func(c *gin.Context, state *State) {
		code := *state
		seelog.Debugf("State: %d", code)
		status, ok := StateMapping[code]
		if !ok {
			seelog.Errorf("Unknown code state: %d", code)
		}

		c.JSON(http.StatusOK, gin.H{
			"code":   code,
			"status": status,
			"data":   CourseMap,
		})
		CourseMap=map[int]map[string][]CourseDetail1{}
		courseList=[]CourseDetail1{}
	}(c, &code)

	// 验证 content type 并解析 body
	contentType := c.Request.Header.Get("Content-Type")
	switch contentType {

	case "application/json":
		if err := c.BindJSON(&dat); err != nil {
			seelog.Error("BindJSON failed, err: %v", err)
		}

	default:
		err = ContentTypeErr
		code = StateContentTypeMismatch
		seelog.Errorf("contentType mismatch: %s", contentType)
		return
	}

	// 验证 identifier 身份
	if dat.Identifier != "terry" {
		code = StateIdentifierMismatch
		seelog.Errorf("identifier mismatch: %s", dat.Identifier)
		return
	}

	// 验证签名
	dataMap := gin.H{
		"identifier": dat.Identifier,
		"data_str":   dat.DataStr,
	}

	paramStr := "4534253453252345353534253453245342"
	var keys []string

	for key := range dataMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		paramStr += key + dataMap[key].(string)
	}

	md5Ctx := md5.New()
	_, err = md5Ctx.Write([]byte( paramStr ))
	if err != nil {
		seelog.Errorf("calc md5 write failed, err: %v", err)
	}

	sign := hex.EncodeToString(md5Ctx.Sum(nil))

	if sign != dat.Signature {
		code = StateSignatureMismatch
		seelog.Errorf("Signature mismatch: recv: %s, calc: %s, paramStr: %s", dat.Signature, sign, paramStr)
		return
	}
	getCourseDayByDay()
}

func getDbConnect1() *sql.DB {
	var err error
	if db1 == nil {
		db1, err = sql.Open("mysql", "root:wananle0@tcp(localhost:3306)/test?charset=utf8")
		if err != nil {
			fmt.Println("连接数据库失败:", err)
			os.Exit(1)
		}
	}
	return db1
}

//查询近x天的课程
func getCourseDayByDay() {
	timeStr := time.Now().Format("2006-01-02")
	t, _ := time.Parse("2006-01-02", timeStr)
	timeToday := t.Unix()-3600*8

	//可以根据请求定制化查多少天的数据
	//先写死成前后5天吧
	fiveDaysAgo := timeToday - 86400*5
	fiveDaysAfter := timeToday + 86400*5
	db1 = getDbConnect1()
	//查询前后二天的课程，每天每天返回
	rows, err := db1.Query("select course_id,url,father_title,subject,`time`,teacher_name,price,child_title,bg_time,teacher_name_detail,add_time from courses where bg_time >= ? and bg_time< ?  group by course_id", fiveDaysAgo, fiveDaysAfter)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		CourseDetail := CourseDetail1{}
		err := rows.Scan(&CourseDetail.CourseId, &CourseDetail.Url, &CourseDetail.FatherTitle, &CourseDetail.Subject, &CourseDetail.Time, &CourseDetail.TeacherName, &CourseDetail.Price,
			&CourseDetail.ChildTitile, &CourseDetail.BgTime, &CourseDetail.TeacherNameDetail, &CourseDetail.AddTime)
		if err != nil {
			log.Fatal(err)
		}
		courseList = append(courseList, CourseDetail)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	var day int
	for i:=0;i<len(courseList);i++{

		if int64(courseList[i].BgTime)-timeToday>=0{
			day=int((int64(courseList[i].BgTime)-timeToday)/86400)
		}else {
			day=int((int64(courseList[i].BgTime)-timeToday)/86400)-1
		}
		if _, ok := CourseMap[day]; !ok {
			CourseMap[day]=make(map[string][]CourseDetail1,0)
		}
		if _, ok := CourseMap[day][courseList[i].Subject]; !ok {
			CourseMap[day][courseList[i].Subject]=make([]CourseDetail1,0)
		}
		CourseMap[day][courseList[i].Subject]=append(CourseMap[day][courseList[i].Subject],courseList[i])
	}
}
func StartSrv() {

	postRouter := gin.Default()
	postRouter.Use(Cors())
	postRouter.POST("/srv_post", srvHandlerPost)
	postRouter.Run(":8000")

}

func main() {
	StartSrv()
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
	}
}
