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
create_date datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP(0),

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
create_date datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP(0),
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
create_date datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP(0),
  PRIMARY KEY ( id ) USING BTREE,
  UNIQUE INDEX  aid ( aid ) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8 COLLATE = utf8_general_ci ROW_FORMAT = Dynamic;`

	prox := `CREATE TABLE if not exists bbd.bbd_ip  (
  id int(11) NOT NULL AUTO_INCREMENT,
  ip varchar(255) CHARACTER SET utf8 COLLATE utf8_general_ci NULL DEFAULT NULL,
  port int(11) NULL DEFAULT 8080,
  status tinyint(4) NOT NULL DEFAULT 1 COMMENT '0 可用 1 待验证',
create_date datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP(0),
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

	var categories = `CREATE TABLE IF NOT EXISTS categories (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,
  category_name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '分类名称',
  created_at timestamp NULL DEFAULT NULL,
  updated_at timestamp NULL DEFAULT NULL,
 UNIQUE INDEX  category_name ( category_name ) USING BTREE,
  PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`

	var topics = `CREATE TABLE IF NOT EXISTS topics (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  av varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '专题在B站唯一标识',
  topic_url varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '专题的地址',
  up_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'up主的唯一标识',
  img varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '专题封面',
  title varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '专题标题',
  description varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '专题描述',
  status tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '状态1可用，0禁用',
  created_at timestamp NULL DEFAULT NULL,
  updated_at timestamp NULL DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY topics_av_unique (av),
  KEY topics_up_id_index (up_id)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`

	var topic_videos = `CREATE TABLE  IF NOT EXISTS  topic_videos (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  topic_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '专辑id',
  img varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '视频封面',
  av varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '视频在B站的标识',
  p varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1' COMMENT '在专辑里面的页码',
  title varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '视频标题',
  category_id int(10) unsigned NOT NULL DEFAULT '1' COMMENT '视频的分类',
  description varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '视频描述',
  view_number int(10) unsigned NOT NULL DEFAULT '1' COMMENT '观看次数',
  status tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '状态1可用，0禁用',
  created_at timestamp NULL DEFAULT NULL,
  updated_at timestamp NULL DEFAULT NULL,
  PRIMARY KEY (id),
  KEY topic_videos_topic_id_index (topic_id),
  KEY topic_videos_av_index (av),
  KEY topic_videos_category_id_index (category_id)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`

	create_table(up, db)
	create_table(topic, db)
	create_table(album, db)
	create_table(prox, db)
	create_table(album_failed, db)

	create_table(categories, db)
	create_table(topics, db)
	create_table(topic_videos, db)

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
