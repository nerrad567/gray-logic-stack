find_package(PkgConfig REQUIRED)
pkg_check_modules(SDL2 REQUIRED sdl2)

# Networking libraries (optional — panel works in demo mode without them)
pkg_check_modules(CURL libcurl)
pkg_check_modules(MOSQUITTO libmosquitto)
pkg_check_modules(CJSON libcjson)

add_executable(retropanel_sim main.c)

# Make lv_conf.h findable by LVGL — it looks in the include path
target_include_directories(retropanel_sim PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}
)
target_include_directories(lvgl PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${SDL2_INCLUDE_DIRS}
)
target_include_directories(retropanel_app PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
)

target_include_directories(retropanel_sim PRIVATE ${SDL2_INCLUDE_DIRS})

# Add networking include paths and compile flag when all libs are available
if(CURL_FOUND AND MOSQUITTO_FOUND AND CJSON_FOUND)
    target_include_directories(retropanel_app PUBLIC
        ${CURL_INCLUDE_DIRS}
        ${MOSQUITTO_INCLUDE_DIRS}
        ${CJSON_INCLUDE_DIRS}
    )
    target_compile_definitions(retropanel_app PUBLIC PANEL_HAS_NETWORKING=1)
    message(STATUS "Networking enabled (curl + mosquitto + cJSON found)")
else()
    message(STATUS "Networking disabled — running in demo-only mode")
    message(STATUS "  Install: sudo apt install libcurl4-openssl-dev libmosquitto-dev libcjson-dev")
endif()

# Link order matters for static libs: app -> lvgl -> SDL2 -> networking -> system libs
target_link_libraries(retropanel_sim PRIVATE
    retropanel_app
    lvgl
    ${SDL2_LIBRARIES}
    ${CURL_LIBRARIES}
    ${MOSQUITTO_LIBRARIES}
    ${CJSON_LIBRARIES}
    m
    pthread
)
target_compile_options(retropanel_sim PRIVATE -Wall -Wextra -Wno-unused-parameter)
