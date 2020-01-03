package main

import (
	"fmt"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	. "github.com/kakuilan/kgo"
	"github.com/spf13/viper"
	"html/template"
	"log"
	"net/http"
	"time"
)

var dbmap = make(map[string]string)

var db *gorm.DB

//数据表模型
type TestModel struct {
	// 注意,不要继承gorm.Model
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Age        int8   `json:"age"`
	Sign       string `json:"sign"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

// 设置TestModel的表名为`tests`
func (TestModel) TableName() string {
	return "tests"
}

func init() {
	//open a db connection
	var err error

	//载入配置
	viper.SetConfigName("conf.yaml")
	viper.AddConfigPath("./config")
	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	//db, err = gorm.Open("mysql", "root:12345@/demo?charset=utf8&parseTime=True&loc=Local")
	var conninfos = []string{}
	conninfos = append(conninfos, viper.GetString("database.user"), ":")
	conninfos = append(conninfos, viper.GetString("database.password"), "@tcp(")
	conninfos = append(conninfos, viper.GetString("database.host"), ":")
	conninfos = append(conninfos, viper.GetString("database.port"), ")/")
	conninfos = append(conninfos, viper.GetString("database.dbname"), "?charset=")
	conninfos = append(conninfos, viper.GetString("database.charset"), "&parseTime=True&loc=Local&readTimeout=500ms")
	connstr := KArr.Implode("", conninfos)
	//println(connstr)

	//db, err = gorm.Open("mysql","name:password@ip:port/databasename?charset=utf8mb4&parseTime=True&loc=Local&readTimeout=500ms")
	db, err = gorm.Open("mysql", connstr)
	if err != nil {
		panic("failed to connect database:" + err.Error())
	}
}

func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d/%02d/%02d", year, month, day)
}

// 登录表单
type LoginForm struct {
	User     string `form:"user" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type UserClaims struct {
	Uid   int    `json:"uid"`
	Agent string `json:"agent"`
	jwt.StandardClaims
}

func setupRouter() *gin.Engine {
	// 禁止控制台日志颜色
	gin.DisableConsoleColor()

	// 创建默认的路由中间件
	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// Get user value
	r.GET("/user/:name", func(c *gin.Context) {
		user := c.Params.ByName("name")
		value, ok := dbmap[user]
		if ok {
			c.JSON(http.StatusOK, gin.H{"user": user, "value": value})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user, "status": "no value"})
		}
	})

	// Authorized group (uses gin.BasicAuth() middleware)
	// Same than:
	// authorized := r.Group("/")
	// authorized.Use(gin.BasicAuth(gin.Credentials{
	//	  "foo":  "bar",
	//	  "manu": "123",
	//}))
	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		"foo":  "bar", // user:foo password:bar
		"manu": "123", // user:manu password:123
	}))

	authorized.POST("admin", func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(string)

		// Parse JSON
		var json struct {
			Value string `json:"value" binding:"required"`
		}

		if c.Bind(&json) == nil {
			dbmap[user] = json.Value
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	r.GET("/someJSON", func(c *gin.Context) {
		data := map[string]interface{}{
			"lang": "GO语言",
			"tag":  "<br>",
		}

		// 输出 : {"lang":"GO\u8bed\u8a00","tag":"\u003cbr\u003e"}
		c.AsciiJSON(http.StatusOK, data)
	})

	// 注入模板函数
	r.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
	})
	r.LoadHTMLGlob("templates/*")

	// html渲染
	//router.LoadHTMLFiles("templates/template1.html", "templates/template2.html")
	r.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Main website",
		})
	})

	// 输出js
	r.GET("/getjs", func(c *gin.Context) {
		c.Header("Content-Type", "text/javascript;charset=UTF-8")
		c.HTML(http.StatusOK, "getjs.tmpl", gin.H{
			"now": time.Date(2017, 07, 01, 0, 0, 0, 0, time.UTC),
		})
	})

	// jsonp
	r.GET("/jsonp", func(c *gin.Context) {
		data := map[string]interface{}{
			"foo": "bar",
		}

		// 访问 jsonp?callback=x
		// 将输出：x({\"foo\":\"bar\"})
		c.JSONP(http.StatusOK, data)
	})

	// Multipart/Urlencoded 绑定表单参数
	r.POST("/login", func(c *gin.Context) {
		// 你可以使用显式绑定声明绑定 multipart form：
		// c.ShouldBindWith(&form, binding.Form)
		// 或者简单地使用 ShouldBind 方法自动绑定：
		var form LoginForm
		// 在这种情况下，将自动选择合适的绑定
		// post必须同时发送user/password两个参数才匹配
		if c.ShouldBind(&form) == nil {
			if form.User == "user" && form.Password == "password" {
				c.JSON(200, gin.H{"status": "you are logged in"})
			} else {
				c.JSON(401, gin.H{"status": "unauthorized"})
			}
		}
	})

	// Multipart/Urlencoded 表单-不强制绑定参数
	r.POST("/form_post", func(c *gin.Context) {
		message := c.PostForm("message")
		nick := c.DefaultPostForm("nick", "anonymous") //默认值

		c.JSON(200, gin.H{
			"status":  "posted",
			"message": message,
			"nick":    nick,
		})
	})

	// 提供 unicode 实体
	r.GET("/json", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"html": "<b>Hello, world!</b>",
			"cn":   "<b>你好，世界！</b>",
		})
	})

	// json-提供字面字符
	r.GET("/purejson", func(c *gin.Context) {
		c.PureJSON(200, gin.H{
			"html": "<b>Hello, world!</b>",
			"cn":   "<b>你好，世界！</b>",
		})
	})

	// 获取请求参数
	//POST /query?id=1234&page=1&ids[a]=1234&ids[b]=hello HTTP/1.1
	//Content-Type: application/x-www-form-urlencoded
	//post参数如
	//name=manu&message=this_is_great&names[first]=thinkerou&names[second]=tianou
	r.POST("/query", func(c *gin.Context) {
		id := c.Query("id")
		page := c.DefaultQuery("page", "0")
		name := c.PostForm("name")
		message := c.PostForm("message")

		//映射查询字符串或表单参数(数组)
		ids := c.QueryMap("ids")
		names := c.PostFormMap("names")

		str := fmt.Sprintf("id: %s; page: %s; name: %s; message: %s", id, page, name, message)
		c.JSON(200, gin.H{
			"str":     str,
			"id":      id,
			"page":    page,
			"name":    name,
			"message": message,
			"ids":     ids,
			"names":   names,
		})
	})

	// 使用 SecureJSON 防止 json 劫持。如果给定的结构是数组值，则默认预置 "while(1)," 到响应体。
	// 你也可以使用自己的 SecureJSON 前缀
	// r.SecureJsonPrefix(")]}',\n")
	r.GET("/SecureJSON", func(c *gin.Context) {
		names := []string{"lena", "austin", "foo", "<b>你好，世界！</b>"}
		// 将输出：while(1);["lena","austin","foo"]
		c.SecureJSON(http.StatusOK, names)
	})

	// 上传单文件
	// 为 multipart forms 设置较低的内存限制 (默认是 32 MiB)
	// router.MaxMultipartMemory = 8 << 20  // 8 MiB
	r.POST("/upload", func(c *gin.Context) {
		// 单文件
		file, _ := c.FormFile("file")
		if file == nil {
			c.JSON(200, gin.H{
				"msg": "none upload file",
			})
		} else {
			log.Println(file.Filename)

			// 上传文件至指定目录
			dst := "./" + file.Filename
			err := c.SaveUploadedFile(file, dst)
			if err != nil {
				log.Print(err.Error())
			}

			c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
		}
	})

	// 读取yaml配置
	r.GET("/readyaml", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"all":      viper.AllSettings(),
			"database": viper.GetStringMap("database"),
		})
	})

	// 生成jwt token
	r.GET("/createjwt", func(c *gin.Context) {
		agent := c.GetHeader("User-Agent")
		agent = KStr.Md5(agent, 16)
		secret := viper.GetString("jwt.secret_key")
		ttl := viper.GetInt64("jwt.token_ttl")

		claims := UserClaims{
			100,
			agent,
			jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Second * time.Duration(ttl)).Unix(),
			},
		}

		tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := tokenClaims.SignedString([]byte(secret))
		if err != nil {
			c.JSON(200, gin.H{
				"status": false,
			})
		} else {
			c.JSON(200, gin.H{
				"secret": secret,
				"ttl":    ttl,
				"token":  token,
			})
		}
	})

	// 解析jwt token
	r.GET("/parsejwt", func(c *gin.Context) {
		tokenStr, _ := c.GetQuery("token")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(viper.GetString("jwt.secret_key")), nil
		})
		if err != nil {
			c.JSON(200, gin.H{
				"status": false,
			})
		} else {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				//结果如 {"claims":{"agent":"6518ca917a5b5888","exp":1577950744,"uid":100}}
				c.JSON(200, gin.H{
					"claims": claims,
				})
			} else {
				c.JSON(200, gin.H{
					"status": false,
				})
			}
		}
	})

	// db-新增记录
	r.POST("/dbadd", func(c *gin.Context) {
		name := KStr.Trim(c.PostForm("name"))
		age := KConv.Str2Int8(c.PostForm("age"))
		sign := c.PostForm("sign")
		now := KTime.Time()

		if KConv.IsEmpty(name) {
			c.JSON(200, gin.H{
				"status": false,
				"msg":    "name不能为空",
			})
			return
		}

		newTest := &TestModel{
			Name:       name,
			Age:        age,
			Sign:       sign,
			CreateTime: now,
			UpdateTime: now,
		}
		db.Create(newTest)
		if newTest.ID > 0 {
			c.JSON(200, gin.H{
				"status": true,
				"obj":    newTest,
			})
		} else {
			c.JSON(200, gin.H{
				"status": false,
				"msg":    "新增失败",
			})
		}
	})

	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
