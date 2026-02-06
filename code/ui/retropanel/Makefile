LVGL_DIR := components/lvgl
LVGL_TAG := v9.2.2
BUILD_DIR := build/sdl
ESP_BUILD_DIR := build/esp

.PHONY: sim-build sim-run sim-clean esp-build esp-flash clean setup fonts

# Setup — clone LVGL if missing
setup: $(LVGL_DIR)/src/lvgl.h

$(LVGL_DIR)/src/lvgl.h:
	@echo "Fetching LVGL $(LVGL_TAG)..."
	git clone --depth=1 --branch $(LVGL_TAG) --single-branch \
		https://github.com/lvgl/lvgl.git $(LVGL_DIR)

# SDL Simulator
sim-build: setup
	@mkdir -p $(BUILD_DIR)
	cd $(BUILD_DIR) && cmake ../.. -DCMAKE_BUILD_TYPE=Debug
	cd $(BUILD_DIR) && make -j$$(nproc)

sim-run: sim-build
	./$(BUILD_DIR)/sdl_simulator/retropanel_sim

sim-clean:
	rm -rf $(BUILD_DIR)

# ESP32-S3 (Phase 4)
esp-build:
	cd esp_project && idf.py build

esp-flash:
	cd esp_project && idf.py -p /dev/ttyUSB0 flash monitor

# Font conversion (requires lv_font_conv: npm install -g lv_font_conv)
fonts:
	./tools/font_convert.sh

clean:
	rm -rf build/
