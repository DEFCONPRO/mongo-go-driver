// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/10gen/mongo-go-driver/mongo/connstring"
	"github.com/10gen/mongo-go-driver/mongo/internal/testutil/helpers"
	"github.com/10gen/mongo-go-driver/mongo/model"
	"github.com/10gen/mongo-go-driver/mongo/private/cluster"
	"github.com/stretchr/testify/require"
)

const seedlistTestDir string = "../../../data/initial-dns-seedlist-discovery/"

type seedlistTestCase struct {
	URI     string
	Seeds   []string
	Hosts   []string
	Error   bool
	Options map[string]interface{}
}

func runSeedlistTest(t *testing.T, filename string, test *seedlistTestCase) {
	t.Run(filename, func(t *testing.T) {
		if runtime.GOOS == "windows" && filename == "two-txt-records" {
			t.Skip("Skipping to avoid windows multiple TXT record lookup bug")
		}
		cs, err := connstring.Parse(test.URI)
		if test.Error {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)

		// DNS records may be out of order from the test files ordering
		seeds := buildSet(test.Seeds)
		hosts := buildSet(cs.Hosts)

		require.Equal(t, hosts, seeds)

		testhelpers.VerifyConnStringOptions(t, cs, test.Options)

		// make a cluster from the options
		c, err := cluster.New(
			cluster.WithConnString(cs),
		)

		require.NoError(t, err)
		for _, host := range test.Hosts {
			_, err := getServerByAddress(host, c)
			require.NoError(t, err)
		}
	})

}

// Test case for all connection string spec tests.
func TestInitialDNSSeedlistDiscoverySpec(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "replica_set" || os.Getenv("AUTH") != "noauth" {
		t.Skip("Skipping on non-replica set topology")
	}

	for _, fname := range testhelpers.FindJSONFilesInDir(t, seedlistTestDir) {
		filepath := path.Join(seedlistTestDir, fname)
		content, err := ioutil.ReadFile(filepath)
		require.NoError(t, err)

		var testCase seedlistTestCase
		require.NoError(t, json.Unmarshal(content, &testCase))

		fname = fname[:len(fname)-5]
		runSeedlistTest(t, fname, &testCase)
	}
}

func buildSet(list []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, s := range list {
		set[s] = struct{}{}
	}
	return set
}

func getServerByAddress(address string, c *cluster.Cluster) (*model.Server, error) {
	selectByName := func(_ *model.Cluster, servers []*model.Server) ([]*model.Server, error) {
		for _, s := range servers {
			if s.Addr.String() == address {
				return []*model.Server{s}, nil
			}
		}
		return []*model.Server{}, nil
	}

	selectedServer, err := c.SelectServer(context.Background(), selectByName, nil)
	if err != nil {
		return nil, err
	}
	return selectedServer.Server.Model(), nil
}