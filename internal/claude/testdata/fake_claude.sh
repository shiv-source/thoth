#!/bin/sh
# Fake Claude CLI for manual testing: prints canned stream-json lines and
# appends its argv to argv.txt next to this script.
# The automated tests in client_test.go write their own inline fakes; this
# copy exists so a human can point CLIClient.Bin here to smoke-test Start.
echo "$@" >> "$(dirname "$0")/argv.txt"
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"hi from cli"}]}}'
echo '{"type":"result","subtype":"success","is_error":false,"result":"done"}'
