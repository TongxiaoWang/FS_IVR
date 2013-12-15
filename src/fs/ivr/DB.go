// fs/ivr/ DB

/*
*	Author : Tongxiao
* 	Date : 2013-12-11
 */

package ivr

import (
	"database/sql"
	"fmt"
	"fs/ivr/eventsocket"
	_ "github.com/go-sql-driver/mysql"
)

type Persistor interface {
	Open() error
	Persist(ivrChannel *IVRChannel)
	Close()
}

type DBPersistor struct {
	DBType string
	DBAddr string
	DBUser string
	DBPwd  string
	DBName string
	DB     *sql.DB
}

func NewDBPersistor(dbType, dbAddr, dbUser, dbPwd, dbName string) *DBPersistor {
	persistor := new(DBPersistor)
	persistor.DBType = dbType
	persistor.DBAddr = dbAddr
	persistor.DBUser = dbUser
	persistor.DBPwd = dbPwd
	persistor.DBName = dbName
	return persistor
}

func (persistor *DBPersistor) Open() error {

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", persistor.DBUser, persistor.DBPwd, persistor.DBAddr, persistor.DBName)
	fmt.Println("DSN :", dsn)

	db, err := sql.Open(persistor.DBType, dsn)
	if err != nil {
		fmt.Println("Open database failure for :", err.Error())
		return err
	}

	fmt.Println("Open mysql db ok.")

	persistor.DB = db

	return nil
}

func (persistor *DBPersistor) Persist(ivrChannel *IVRChannel) {
	stmt, _ := persistor.DB.Prepare("insert into IvrNode values(?,?,?,?)")
	uuid, _ := eventsocket.GenUUID()
	stmt.Exec(uuid, ivrChannel.ActiveNode, eventsocket.GetDateTime(), ivrChannel.ChannelId)
	defer stmt.Close()
}

func (persistor *DBPersistor) Close() {
	persistor.DB.Close()
}

func DBDemo(dbAddr, dbUser, dbPwd, dbName string) {
	//[username[:password]@][protocol[(address)]]/dbname[?param1=value1&paramN=valueN]
	dsn := fmt.Sprintf("%s:%s@%s/%s?charset=utf8", dbUser, dbPwd, dbAddr, dbName)
	fmt.Println("DSN :", dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("Open database failure for :", err.Error())
		return
	}

	fmt.Println("Open mysql db ok.")

	stmt, err := db.Prepare("insert into IvrNode values(?,?,?,?)")
	if err != nil {
		fmt.Println("Insert data failure for :", err.Error())
		return
	}

	_, err = stmt.Exec("1321442332", "root", "2013-12-11 22:00:07", "658998")
	if err != nil {
		fmt.Println("Insert data failure for :", err.Error())
		return
	}

	fmt.Println("Insert data ok.")

}
