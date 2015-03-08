#!/bin/sh
#
# Copyright (c) 2015 Matt Bell
# MIT Licensed; see the LICENSE file in this repository.
#

test_description="Test HTTP Gateway"

. lib/test-lib.sh

test_init_ipfs
test_config_ipfs_gateway_readonly $ADDR_GWAY
test_launch_ipfs_daemon

port=$PORT_GWAY
apiport=$PORT_API

# TODO check both 5001 and 5002.
# 5001 should have a readable gateway (part of the API)
# 5002 should have a readable gateway (using ipfs config Addresses.Gateway)
# but ideally we should only write the tests once. so maybe we need to
# define a function to test a gateway, and do so for each port.
# for now we check 5001 here as 5002 will be checked in gateway-writable.

test_expect_success "GET IPFS path succeeds" '
  echo "Hello Worlds!" >expected &&
  HASH=$(ipfs add -q expected) &&
  curl -sfo actual "http://127.0.0.1:$port/ipfs/$HASH"
'

test_expect_success "GET IPFS path output looks good" '
  test_cmp expected actual &&
  rm actual
'

test_expect_success "GET IPFS directory path succeeds" '
  mkdir dir &&
  echo "12345" >dir/test &&
  ipfs add -r -q dir >actual &&
  HASH2=$(tail -n 1 actual) &&
  curl -sf "http://127.0.0.1:$port/ipfs/$HASH2"
'

test_expect_success "GET IPFS directory file succeeds" '
  curl -sfo actual "http://127.0.0.1:$port/ipfs/$HASH2/test"
'

test_expect_success "GET IPFS directory file output looks good" '
  test_cmp dir/test actual
'

test_expect_failure "GET IPNS path succeeds" '
  ipfs name publish "$HASH" &&
  NAME=$(ipfs config Identity.PeerID) &&
  curl -sfo actual "http://127.0.0.1:$port/ipns/$NAME"
'

test_expect_failure "GET IPNS path output looks good" '
  test_cmp expected actual
'

test_expect_success "GET invalid IPFS path errors" '
  test_must_fail curl -sf "http://127.0.0.1:$port/ipfs/12345"
'

test_expect_success "GET invalid path errors" '
  test_must_fail curl -sf "http://127.0.0.1:$port/12345"
'

test_expect_success "GET /webui returns code expected" '
  curl -I http://127.0.0.1:$apiport/webui >actual &&
  RESP=$(head -1 actual) &&
  (expr "$RESP" : "HTTP/1.1 302 Found\s" ||
   expr "$RESP" : "HTTP/1.1 301 Moved Permanently\s")
'

test_expect_success "GET /webui/ returns code expected" '
  curl -I http://127.0.0.1:$apiport/webui/ > actual &&
  RESP=$(head -1 actual) &&
  (expr "$RESP" : "HTTP/1.1 302 Found\s" ||
   expr "$RESP" : "HTTP/1.1 301 Moved Permanently\s")
'

test_kill_ipfs_daemon

test_done
