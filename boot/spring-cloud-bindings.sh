if [[ "${BPL_SPRING_CLOUD_BINDINGS_ENABLED:=y}" == "y" ]]; then
    printf "Spring Cloud Bindings Boot Auto-Configuration Enabled\n"
    export JAVA_TOOL_OPTIONS="${JAVA_TOOL_OPTIONS} -Dorg.springframework.cloud.bindings.boot.enable=true"
fi
