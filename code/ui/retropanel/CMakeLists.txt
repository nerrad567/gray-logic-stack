cmake_minimum_required(VERSION 3.16)
project(retropanel C)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# LVGL source files
set(LVGL_DIR ${CMAKE_CURRENT_SOURCE_DIR}/components/lvgl)

# Collect LVGL sources (core library)
file(GLOB_RECURSE LVGL_SOURCES
    ${LVGL_DIR}/src/*.c
)

# Our application sources
file(GLOB_RECURSE APP_SOURCES
    ${CMAKE_CURRENT_SOURCE_DIR}/src/*.c
)

# Build LVGL as a static library
add_library(lvgl STATIC ${LVGL_SOURCES})
target_include_directories(lvgl PUBLIC
    ${LVGL_DIR}
)
# lv_conf.h is provided per-target (SDL or ESP) — the simulator
# directory is added by sdl_simulator/CMakeLists.txt
target_compile_options(lvgl PRIVATE -w)  # Suppress LVGL internal warnings

# Application library (shared between SDL and ESP targets)
add_library(retropanel_app STATIC ${APP_SOURCES})
target_include_directories(retropanel_app PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/src
    ${LVGL_DIR}
)
target_link_libraries(retropanel_app PUBLIC lvgl)

# Build SDL simulator by default
add_subdirectory(sdl_simulator)
