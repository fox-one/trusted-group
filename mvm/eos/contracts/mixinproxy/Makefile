all : mixinproxy.wasm

mixinproxy.wasm : tables.go mixinproxy.go
	eosio-go build -o mixinproxy.wasm

test : mixinproxy.wasm
	run-ipyeos -m pytest -x -s tests/test.py
#	run-ipyeos -m pytest -x -s tests/test.py -k test_event

testdebug : mixinproxy.wasm
	eosio-go build -tags debug -o mixinproxy.wasm .
	run-ipyeos -m pytest -x -s tests/test.py -k test_debug

