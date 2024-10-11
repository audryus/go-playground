package main

import (
	"os"
	"time"

	"github.com/gocql/gocql"
	"github.com/scylladb/gocqlx"
	"github.com/scylladb/gocqlx/qb"
	"github.com/scylladb/gocqlx/table"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var stmts = createStatements()

func main() {
	logger := CreateLogger("info")
	cluster := CreateCluster(gocql.Quorum, "todpoint", "localhost")
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		logger.Fatal("unable to connect to scylla", zap.Error(err))
	}
	session.ExecuteBatch(session.NewBatch(gocql.UnloggedBatch))

	defer session.Close()
	SelectQuery(session, logger)
	insertQuery(session, "Mike", "Tyson", "RECEIVED", "url", "http://www.facebook.com/mtyson", time.Now(), logger)
	insertQuery(session, "Alex", "Jones", "RECEIVED", "url", "http://www.facebook.com/ajones", time.Now(), logger)
	SelectQuery(session, logger)
	deleteQuery(session, "Mike", logger)
	SelectQuery(session, logger)
	deleteQuery(session, "Alex", logger)
	SelectQuery(session, logger)

}

func SelectQuery(session *gocql.Session, logger *zap.Logger) {
	logger.Info("Displaying Results:" + time.Now().String())
	var rs []Record
	err := gocqlx.Query(session.Query(stmts.sel.stmt), stmts.sel.names).SelectRelease(&rs)
	if err != nil {
		logger.Warn("select catalog.mutant", zap.Error(err))
		return
	}
	for _, r := range rs {
		logger.Info("\t" + r.Id + " " + r.HashCode + ", " + r.Status + ", " + r.Kind + ", " + r.Url + ", " + r.TsCriacao.String())
	}
}

func insertQuery(session *gocql.Session, id, hashCode, status, kind, url string, tsCriacao time.Time, logger *zap.Logger) {
	logger.Info("Inserting " + id + "......")
	r := Record{
		Id:        id,
		HashCode:  hashCode,
		Status:    status,
		Url:       url,
		Kind:      kind,
		TsCriacao: tsCriacao,
	}
	err := gocqlx.Query(session.Query(stmts.ins.stmt), stmts.ins.names).BindStruct(r).ExecRelease()
	if err != nil {
		logger.Error("insert todpoint.url", zap.Error(err))
	}
}

func deleteQuery(session *gocql.Session, id string, logger *zap.Logger) {
	logger.Info("Deleting " + id + "......")
	r := Record{
		Id: id,
	}
	err := gocqlx.Query(session.Query(stmts.del.stmt), stmts.del.names).BindStruct(r).ExecRelease()
	if err != nil {
		logger.Error("delete todpoint.url", zap.Error(err))
	}
}

func CreateCluster(consistency gocql.Consistency, keyspace string, hosts ...string) *gocql.ClusterConfig {
	retryPolicy := &gocql.ExponentialBackoffRetryPolicy{
		Min:        time.Second,
		Max:        10 * time.Second,
		NumRetries: 5,
	}
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Timeout = 5 * time.Second
	cluster.RetryPolicy = retryPolicy
	cluster.Consistency = consistency
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	return cluster
}

func CreateLogger(level string) *zap.Logger {
	lvl := zap.NewAtomicLevel()
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl.SetLevel(zap.InfoLevel)
	}
	encoderCfg := zap.NewDevelopmentEncoderConfig()

	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		lvl,
	))

	return logger
}

func createStatements() *statements {
	m := table.Metadata{
		Name:    "url",
		Columns: []string{"id", "hash_code", "status", "kind", "url", "ts_criacao"},
		PartKey: []string{"id"},
	}
	tbl := table.New(m)

	deleteStmt, deleteNames := tbl.Delete()
	insertStmt, insertNames := tbl.Insert()
	// Normally a select statement such as this would use `tbl.Select()` to select by
	// primary key but now we just want to display all the records...
	selectStmt, selectNames := qb.Select(m.Name).Columns(m.Columns...).ToCql()
	return &statements{
		del: query{
			stmt:  deleteStmt,
			names: deleteNames,
		},
		ins: query{
			stmt:  insertStmt,
			names: insertNames,
		},
		sel: query{
			stmt:  selectStmt,
			names: selectNames,
		},
	}
}

type query struct {
	stmt  string
	names []string
}

type statements struct {
	del query
	ins query
	sel query
}

type Record struct {
	Id        string    `db:"id"`
	HashCode  string    `db:"hash_code"`
	Status    string    `db:"status"`
	Kind      string    `db:"kind"`
	Url       string    `db:"url"`
	TsCriacao time.Time `db:"ts_criacao"`
}
