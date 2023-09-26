package common

import (
    "database/sql"
    "errors"

    _ "github.com/mattn/go-sqlite3"
)

type ClipContentInfo struct {
    ClientID string
    Username string
    Content  string
}

type DBInfo struct {
    dbFile string
    conn   *sql.DB
}

//InitDB init sqlite database with specify file
func InitDB(dbFile string) *DBInfo {
    conn, err := sql.Open("sqlite3", dbFile)
    if err != nil {
        return nil
    }

    return &DBInfo{
        dbFile: dbFile,
        conn:   conn,
    }
}

//Close close the sqlite database
func (db *DBInfo) Close() {
    if db.conn != nil {
        db.conn.Close()
    }
}

func (db *DBInfo) createSQL(sql string) error {
    if db.conn == nil {
        return errors.New("sqlite is not init")
    }

    _, err := db.conn.Exec(sql)
    return err
}

func (db *DBInfo) CreateUserInfoTable() error {

    // create userinfo table if not exist
    sql_table := `
    CREATE TABLE IF NOT EXISTS userinfo(
        uid INTEGER PRIMARY KEY AUTOINCREMENT,
        username VARCHAR(64) UNIQUE NOT NULL,
        password VARCHAR(64) NOT NULL
    );
    `
    return db.createSQL(sql_table)
}

// insert or update
func (db *DBInfo) InsertUserInfo(auths []AuthConfig) error {
    if db.conn == nil {
        return errors.New("sqlite is not init")
    }

    stmt, err := db.conn.Prepare("REPLACE INTO userinfo(username, password) values(?, ?)")
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, auth := range auths {
        _, err := stmt.Exec(auth.User, auth.Password)
        if err != nil {
            return err
        }
    }

    return nil
}

func (db *DBInfo) GetUserByName(username string) *AuthConfig {
    if db.conn == nil {
        return nil
    }

    auth := AuthConfig{}
    err := db.conn.QueryRow("SELECT username, password FROM userinfo WHERE username = ?", username).Scan(&auth.User, &auth.Password)
    if err != nil {
        return nil
    } else {
        return &auth
    }
}

func (db *DBInfo) GetPassword(username string) string {
    auth := db.GetUserByName(username)
    if auth == nil {
        return ""
    }

    return auth.Password
}

func (db *DBInfo) GetUsers() []AuthConfig {
    if db.conn == nil {
        return nil
    }

    rows, err := db.conn.Query("SELECT username, password FROM userinfo;")
    if err != nil {
        return nil
    }

    var auths []AuthConfig
    for rows.Next() {
        auth := AuthConfig{}
        err = rows.Scan(&auth.User, &auth.Password)
        if err != nil {
            continue
        } else {
            auths = append(auths, auth)
        }
    }

    return auths
}

func (db *DBInfo) CreateContentInfoTable() error {

    // create content info table if not exist
    sql_table := `
    CREATE TABLE IF NOT EXISTS contentinfo(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        clientid VARCHAR(64) NOT NULL,
        username VARCHAR(64) NOT NULL,
        content VARCHAR(64) NOT NULL,
        timestamp DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now', 'localtime')),
        UNIQUE (clientid, username)
    );
    `

    return db.createSQL(sql_table)
}

func (db *DBInfo) InsertClipContent(content *ClipContentInfo) error {
    if db.conn == nil {
        return errors.New("sqlite is not init")
    }

    stmt, err := db.conn.Prepare("REPLACE INTO contentinfo(clientid, username, content) values(?, ?, ?)")
    if err != nil {
        return err
    }
    defer stmt.Close()

    _, err = stmt.Exec(content.ClientID, content.Username, content.Content)
    return err
}

func (db *DBInfo) GetClipContentByName(username string) string {
    if db.conn == nil {
        return ""
    }

    var content string
    err := db.conn.QueryRow("SELECT content FROM contentinfo WHERE username = ? ORDER BY timestamp DESC", username).Scan(&content)
    if err != nil {
        return ""
    } else {
        return content
    }
}

func (db *DBInfo) GetClipContentByID(clientid string) string {
    if db.conn == nil {
        return ""
    }

    var content string
    err := db.conn.QueryRow("SELECT content FROM contentinfo WHERE clientid = ? ORDER BY timestamp DESC", clientid).Scan(&content)
    if err != nil {
        return ""
    } else {
        return content
    }
}

func (db *DBInfo) GetClipContents() []ClipContentInfo {
    if db.conn == nil {
        return nil
    }

    rows, err := db.conn.Query("SELECT clientid, username, content FROM contentinfo;")
    if err != nil {
        return nil
    }

    var clips []ClipContentInfo
    for rows.Next() {
        clip := ClipContentInfo{}
        err = rows.Scan(&clip.ClientID, &clip.Username, &clip.Content)
        if err != nil {
            continue
        } else {
            clips = append(clips, clip)
        }
    }

    return clips
}
