package main

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	addr := "localhost:2379"

	endpoints := []string{addr}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		panic(err)
	}
	ctx, timeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer timeout()

	_, err = cli.Put(ctx, "chave", "valor 1233")
	if err != nil {
		panic(err)
	}

	resp, err := cli.Get(ctx, "chave")
	if err != nil {
		panic(err)
	}

	fmt.Printf("valor retornado: %s", resp.Kvs[0].Value)

}
