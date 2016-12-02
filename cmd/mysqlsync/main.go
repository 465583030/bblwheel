package main

import (
	"context"
	"os"

	"flag"

	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
)

var (
	sid    int
	flavor string
	host   string
	port   int
	user   string
	passwd string
)

func init() {
	flag.IntVar(&sid, "sid", 100, "ServerID")
	flag.StringVar(&flavor, "flavor", "mysql", "")
	flag.StringVar(&host, "host", "127.0.0.1", "source mysql host address")
	flag.IntVar(&port, "port", 3306, "source mysql host port")
	flag.StringVar(&user, "user", "root", "source mysql user name")
	flag.StringVar(&passwd, "passwd", "", "source mysql user password")
}
func main() {
	flag.Parse()
	// Create a binlog syncer with a unique server id, the server id must be different from other MySQL's.
	// flavor is mysql or mariadb
	cfg := replication.BinlogSyncerConfig{
		ServerID: uint32(sid),
		Flavor:   flavor,
		Host:     host,
		Port:     uint16(port),
		User:     user,
		Password: passwd,
	}
	syncer := replication.NewBinlogSyncer(&cfg)

	// Start sync with sepcified binlog file and position
	streamer, _ := syncer.StartSync(mysql.Position{})

	// or you can start a gtid replication like
	// streamer, _ := syncer.StartSyncGTID(gtidSet)
	// the mysql GTID set likes this "de278ad0-2106-11e4-9f8e-6edd0ca20947:1-2"
	// the mariadb GTID set likes this "0-1-100"

	for {
		ev, _ := streamer.GetEvent(context.Background())
		// Dump event
		ev.Dump(os.Stdout)

	}

	// // or we can use a timeout context
	// for {
	// 	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	// 	e, _ := s.GetEvent(ctx)
	// 	cancel()

	// 	if err == context.DeadlineExceeded {
	// 		// meet timeout
	// 		continue
	// 	}

	// 	ev.Dump(os.Stdout)
	// }
}
