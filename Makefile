MODULE_NAME=nss_dnd
GO_SO=lib$(MODULE_NAME).so
FINAL_SO=lib$(MODULE_NAME).so.2
HEADER_SO=lib$(MODULE_NAME).h
TARGET_DIR=/lib/$(shell uname -m)-linux-gnu

.PHONY: build install clean

# Step 1: Build Go shared object
$(GO_SO): $(MODULE_NAME).go
	go build -buildmode=c-shared -o $(GO_SO) $(MODULE_NAME).go

# Step 2: Build final glibc-compatible NSS module
$(FINAL_SO): $(MODULE_NAME).c $(GO_SO)
	gcc -fPIC -shared -Wl,-soname,$(FINAL_SO) -o $(FINAL_SO) $(MODULE_NAME).c $(GO_SO)

build: $(FINAL_SO)

# Install to NSS path (change to /usr/lib if needed)
install: $(FINAL_SO)
	sudo cp $(GO_SO) $(FINAL_SO) $(TARGET_DIR)/
	sudo ldconfig

clean:
	rm -f $(GO_SO) $(FINAL_SO) $(HEADER_SO)
