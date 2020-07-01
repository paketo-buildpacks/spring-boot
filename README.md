# `gcr.io/paketo-buildpacks/spring-boot`
The Paketo Spring Boot Buildpack is a Cloud Native Buildpack that contributes Spring Boot dependency information and slices an application into multiple layers.

## Behavior
This buildpack will participate if all the following conditions are met

* `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Version` entry

The buildpack will do the following:

* Contributes Spring Boot version to `org.springframework.boot.version` image label
* Contributes Spring Boot configuration metadata to `org.springframework.boot.spring-configuration-metadata.json` image label
* Contributes `Implementation-Title` manifest entry to `org.opencontainers.image.title` image label
* Contributes `Implementation-version` manifest entry to `org.opencontainers.image.version` image label
* Contributes dependency information extracted from Maven naming conventions to the image's BOM
* If `<APPLICATION_ROOT>/META-INF/dataflow-configuration-metadata-whitelist.properties` exists
  * Contributes Spring Cloud Data Flow configuration metadata to `org.springframework.cloud.dataflow.spring-configuration-metadata.json` image label
* If `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Layers-Index` entry
  * Contributes application slices as defined by the layer indexes
* `$BP_BOOT_NATIVE_IMAGE` is set
  * Creates a GraalVM native image and removes existing bytecode.
* If the application is a reactive web application
  * Configures `$BPL_JVM_THREAD_COUNT` to 50

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_BOOT_NATIVE_IMAGE` | Whether to build a native image from the application.  Defaults to false.
| `$BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS` | Configure the arguments to pass to native image build

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
