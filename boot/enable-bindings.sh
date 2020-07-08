if [[ "${BPL_SPRING_CLOUD_BINDINGS_ENABLED:=y}" == "y" ]]; then
    printf "Spring Cloud Bindings Boot Auto-Configuration Enabled\n"
    export JAVA_OPTS="${JAVA_OPTS} -Dorg.springframework.cloud.bindings.boot.enable=true"
fi
