## Repository information

This repository contains the built'n session databases for the [Q web framework](https://github.com/kataras/q).

## How to Register?



```go
//...
import "github.com/kataras/q/sessiondb/$FOLDER"
//...

db := $FOLDER.New($FOLDER.Config{configuration_here})

//...
	q.Q{
		//...
		Session: q.Session{Cookie: "mysessionid", Expires: 4 * time.Hour, GcDuration: 2 * time.Hour, Databases: q.Databases{db}},
		//...
		}.Go()
//...
```

> Note: You can use more than one database to save the session values, but the initial data will come from the first non-empty `Load`, look inside [code](https://github.com/kataras/q/sessiondb/blob/master/redis/database.go) for more information on how to create your own session database.
