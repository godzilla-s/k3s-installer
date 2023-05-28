package node

import "sync"

type clusterToken struct {
	token string
	sync.Once
}

var token clusterToken

func setToken(tokenstr string) {
	token.Do(func() {
		token.token = tokenstr
	})
}

func getToken() string {
	return token.token
}
