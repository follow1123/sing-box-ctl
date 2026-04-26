#!/bin/bash

binPath="$SINGBOX_SERVICE_BINARY"
workingDir="$SINGBOX_SERVICE_WORKING_DIR"
configPath="$SINGBOX_SERVICE_CONFIG_PATH"
action="$SINGBOX_SERVICE_ACTION"
highPermission="$SINGBOX_SERVICE_HIGH_PERMISSION"

# Check if the service process is running
ServiceProcess() {
    pgrep sing-box
}

# Start the service
ServiceStart() {
    if ServiceProcess >/dev/null; then
        return
    fi

    if [ "$highPermission" == "1" ]; then
        echo "Administrator privileges are required to start the service."
                exit 1
        else
        nohup "$binPath" -D "$workingDir" -c "$configPath" run > /dev/null 2>&1 &
    fi
}

# Stop the service
ServiceStop() {
    pid=$(ServiceProcess)
    if [ -n "$pid" ]; then
        kill "$pid"
        if [ $? -ne 0 ]; then
            kill -9 "$pid"
        fi
        TurnOffSystemProxy
    fi
}

# Turn off system proxy
TurnOffSystemProxy() {
    # todo
        true
}

# Main logic
case "${action,,}" in
    start)
        ServiceStart
        sleep 0.5
        if ServiceProcess >/dev/null; then
            exit 0
        else
            exit 1
        fi
        ;;
    stop)
        ServiceStop
        sleep 0.5
        if ServiceProcess >/dev/null; then
            exit 1
        else
            exit 0
        fi
        ;;
    restart)
        ServiceStop
        sleep 0.5
        ServiceStart
        sleep 0.5
        if ServiceProcess >/dev/null; then
            exit 0
        else
            exit 1
        fi
        ;;
    is_running)
        if ServiceProcess >/dev/null; then
            echo "true"
        else
            echo "false"
        fi
        ;;
    *)
        echo "Unknown action: $action"
        exit 1
        ;;
esac
