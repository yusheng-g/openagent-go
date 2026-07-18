.PHONY: all tool observer clean

PLUGINS_DIR := ./plugins

all: tool observer

tool:
	cd tool && cargo +stable build --release --target wasm32-unknown-unknown
	cp tool/target/wasm32-unknown-unknown/release/example_agent_tool.wasm $(PLUGINS_DIR)/echo.wasm

observer:
	cd observer && cargo +stable build --release --target wasm32-unknown-unknown
	cp observer/target/wasm32-unknown-unknown/release/example_agent_observer.wasm $(PLUGINS_DIR)/logger.wasm

clean:
	rm -f $(PLUGINS_DIR)/*.wasm
	cd tool && cargo clean
	cd observer && cargo clean
