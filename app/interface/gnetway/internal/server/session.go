// Copyright (c) 2021-present,  Teamgram Studio (https://teamgram.io).
//  All rights reserved.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package server

import (
	"errors"
	"os"
	"strings"

	"github.com/teamgram/teamgram-server/app/interface/gnetway/internal/config"
	sessionclient "github.com/teamgram/teamgram-server/app/interface/session/client"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/hash"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/netx"
	"github.com/zeromicro/go-zero/core/stringx"
	"github.com/zeromicro/go-zero/zrpc"
)

const (
	allEths  = "0.0.0.0"
	envPodIp = "POD_IP"
)

var (
	ErrSessionNotFound = errors.New("not found session")
)

func figureOutListenOn(listenOn string) string {
	fields := strings.Split(listenOn, ":")
	if len(fields) == 0 {
		return listenOn
	}

	host := fields[0]
	if len(host) > 0 && host != allEths {
		return listenOn
	}

	ip := os.Getenv(envPodIp)
	if len(ip) == 0 {
		ip = netx.InternalIp()
	}
	if len(ip) == 0 {
		return listenOn
	}

	return strings.Join(append([]string{ip}, fields[1:]...), ":")
}

type Session struct {
	gatewayId   string
	dispatcher  *hash.ConsistentHash
	errNotFound error
	sessions    map[string]sessionclient.SessionClient
}

func NewSession(c config.Config) *Session {
	sess := &Session{
		gatewayId:   figureOutListenOn(c.ListenOn),
		dispatcher:  hash.NewConsistentHash(),
		errNotFound: ErrSessionNotFound,
		sessions:    make(map[string]sessionclient.SessionClient),
	}
	sess.watch(c.Session)

	return sess
}

func (sess *Session) watch(c zrpc.RpcClientConf) {
	sub, _ := discov.NewSubscriber(c.Etcd.Hosts, c.Etcd.Key)
	update := func() {
		values := sub.Values()
		if len(values) == 0 {
			return
		}

		var (
			addClis    []sessionclient.SessionClient
			removeClis []sessionclient.SessionClient
		)

		sessions := map[string]sessionclient.SessionClient{}
		for _, v := range values {
			if old, ok := sess.sessions[v]; ok {
				sessions[v] = old
				continue
			}
			c.Endpoints = []string{v}
			cli, err := zrpc.NewClient(c)
			if err != nil {
				logx.Error("watchComet NewClient(%+v) error(%v)", values, err)
				return
			}
			sessionCli := sessionclient.NewSessionClient(cli)
			sessions[v] = sessionCli

			addClis = append(addClis, sessionCli)
		}

		for key, old := range sess.sessions {
			if !stringx.Contains(values, key) {
				removeClis = append(removeClis, old)
			}
		}

		for _, n := range addClis {
			sess.dispatcher.Add(n)
		}

		for _, n := range removeClis {
			sess.dispatcher.Remove(n)
		}

		sess.sessions = sessions
	}

	sub.AddListener(update)
	update()
}

//func (sess *Session) getSessionClient(key string) (sessionclient.SessionClient, error) {
//	val, ok := sess.dispatcher.Get(key)
//	if !ok {
//		return nil, ErrSessionNotFound
//	}
//
//	return val.(sessionclient.SessionClient), nil
//}

func (sess *Session) invokeByKey(key string, cb func(client sessionclient.SessionClient) (err error)) error {
	val, ok := sess.dispatcher.Get(key)
	if !ok {
		return ErrSessionNotFound
	}

	if cb == nil {
		return nil
	}

	return cb(val.(sessionclient.SessionClient))
}
