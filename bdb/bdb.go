package bdb

import (
	"../config"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"sync"
)

var globalDb *sql.DB
var lock sync.RWMutex
var isOK = false

func init() {
	var once sync.Once
	once.Do(initDB)
}

func initDB() {
	lock.Lock()
	isOK, globalDb = openDB()
	lock.Unlock()
}
func Close() {
	globalDb.Close()
}
func GlobalIsOK() bool {
	lock.Lock()
	defer lock.Unlock()
	return isOK
}
func GlobalDB() *sql.DB {
	lock.Lock()
	defer lock.Unlock()
	return globalDb
}
func openDB() (success bool, db *sql.DB) {
	if globalDb != nil {
		return true, globalDb
	}
	var isOpen bool
	db, err := sql.Open("mysql", config.DB_Driver)
	if err != nil {
		panic(err)
		isOpen = false
	} else {
		isOpen = true
	}
	var up = `
CREATE TABLE IF NOT EXISTS bbd.bbd_up  (
  id int(11) NOT NULL AUTO_INCREMENT,
  mid int(11) NULL DEFAULT NULL,
  status tinyint(4) NOT NULL DEFAULT 0,
 face  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
   name  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
  PRIMARY KEY (id) USING BTREE,
  UNIQUE INDEX mid (mid) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;
`
	var topic = `CREATE TABLE IF NOT EXISTS  bbd.bbd_topic  (
   id  int(11) NOT NULL AUTO_INCREMENT,
   mid  int(11) NOT NULL,
   aid  int(11) NOT NULL,
   title  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
   pic  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
   description  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
   status  tinyint(4) NOT NULL DEFAULT 0,
  PRIMARY KEY ( id ) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;`
	var album = `CREATE TABLE IF NOT EXISTS bbd.bbd_album  (
   id  int(11) NOT NULL AUTO_INCREMENT,
   videos  int(11) NOT NULL DEFAULT 1,
   title  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
   state  tinyint(4) NULL DEFAULT 0 COMMENT '三方系统，与我们无关',
   originTitle  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
   origin  json NULL,
   aid  int(11) NOT NULL,
status  tinyint(4) NOT NULL DEFAULT 0,
  PRIMARY KEY ( id ) USING BTREE,
  UNIQUE INDEX  aid ( aid ) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;`

	prox := `CREATE TABLE if not exists bbd.bbd_ip  (
  id int(11) NOT NULL AUTO_INCREMENT,
  ip varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
  port int(11) NULL DEFAULT 8080,
  status tinyint(4) NOT NULL DEFAULT 1 COMMENT '0 可用 1 待验证',
  PRIMARY KEY (id) USING BTREE,
  UNIQUE INDEX ip(ip) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;`

	var album_failed = `CREATE TABLE if not exists bbd.bbd_album_failed   (
   id  int(11) NOT NULL AUTO_INCREMENT,
   album_url  varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL,
   status  tinyint(4) NOT NULL DEFAULT 0,
  PRIMARY KEY ( id ) USING BTREE,
  UNIQUE INDEX  album_url ( album_url ) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;`
	create_table(up, db)
	create_table(topic, db)
	create_table(album, db)
	create_table(prox, db)
	create_table(album_failed, db)

	return isOpen, db
}

func create_table(sql string, db *sql.DB) {
	stmt, err := db.Prepare(sql)
	if err != nil {
		db.Close()
		panic(err)
	}
	stmt.Exec()
	stmt.Close()
}
