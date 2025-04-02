# Paketo Buildpack for Spring Boot

## Buildpack ID: `paketo-buildpacks/spring-boot`
## Registry URLs: `docker.io/paketobuildpacks/spring-boot`

The Paketo Buildpack for Spring Boot is a Cloud Native Buildpack that contributes Spring Boot dependency information and slices an application into multiple layers.

## Behavior

This buildpack will always detect.

This buildpack will participate at build time if all the following conditions are met:

* `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Version` entry

The buildpack will do the following:

* Contributes Spring Boot version to `org.springframework.boot.version` image label
* If `<APPLICATION_ROOT>/META-INF/dataflow-configuration-metadata.properties` exists
  * Contributes Spring Boot configuration metadata to `org.springframework.boot.spring-configuration-metadata.json` image label
  * Contributes Spring Cloud Data Flow configuration metadata to `org.springframework.cloud.dataflow.spring-configuration-metadata.json` image label
* Contributes `Implementation-Title` manifest entry to `org.opencontainers.image.title` image label
* Contributes `Implementation-version` manifest entry to `org.opencontainers.image.version` image label
* Contributes dependency information extracted from Maven naming conventions to the image's BOM
* When contributing to a JVM application:
    * Contributes [Spring Cloud Bindings][b] as an application dependency
      * This enables bindings-aware Spring Boot auto-configuration when [CNB bindings][c] are present during launch
      * The version of the Spring Cloud Bindings library to install will be determined by (in order):
        * An explicit value set in `BP_SPRING_CLOUD_BINDINGS_VERSION` by the user
        * The Spring Boot version from  `<APPLICATION_ROOT>/META-INF/MANIFEST.MF`: Boot 2.x will install Spring Cloud Bindings v1, Boot 3.x will install Spring Cloud Bindings v2
    * If `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Layers-Index` entry
      * Contributes application slices as defined by the layer's index
    * If the application is a reactive web application
      * Configures `$BPL_JVM_THREAD_COUNT` to 50
    * If the application is AOT instrumented (presence of `META-INF/native-image` folder) AND `BP_SPRING_AOT_ENABLED` is set to `true`
      * set `BPL_SPRING_AOT_ENABLED` to true
      * add `-Dspring.aot.enabled=true` to `JAVA_TOOL_OPTIONS` at runtime
    * If the application is AOT instrumented (presence of `META-INF/native-image` folder) AND `BP_SPRING_AOT_ENABLED` is set to `true` AND `CDS_TRAINING_JAVA_TOOL_OPTIONS` is set
      * fail the build with "build failed because of invalid user configuration" - the reason being is that the AOT classes used during training run won't be compatible with a different set of `JAVA_TOOL_OPTIONS` at runtime
      * the Spring team explains this issue in detail here: https://github.com/spring-projects/spring-boot/issues/41348
* If `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Native-Processed` entry OR if `$BP_MAVEN_ACTIVE_PROFILES` contains the `native` profile:
  * A build plan entry is provided, `native-image-application`, which can be required by the `native-image` [buildpack](https://github.com/paketo-buildpacks/native-image) to automatically trigger a native image build
* When contributing to a native image application:
   * Adds classes from the executable JAR and entries from `classpath.idx` to the build-time class path, so they are available to `native-image`

[b]: https://github.com/spring-cloud/spring-cloud-bindings
[c]: https://github.com/buildpacks/spec/blob/main/extensions/bindings.md

## Configuration
| Environment Variable                  | Description                                                                                                                                                                                                                                                             |
|---------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `$BP_SPRING_CLOUD_BINDINGS_DISABLED`  | Whether to contribute Spring Cloud Bindings support to the image at build time.  Defaults to false.                                                                                                                                                                     |
| `$BPL_SPRING_CLOUD_BINDINGS_DISABLED` | Whether to auto-configure Spring Boot environment properties from bindings at runtime. This requires Spring Cloud Bindings to have been installed at build time or it will do nothing. Defaults to false.                                                               |
| `$BPL_SPRING_CLOUD_BINDINGS_ENABLED`  | Deprecated in favour of `$BPL_SPRING_CLOUD_BINDINGS_DISABLED`. Whether to auto-configure Spring Boot environment properties from bindings at runtime. This requires Spring Cloud Bindings to have been installed at build time or it will do nothing. Defaults to true. |
| `$BP_SPRING_CLOUD_BINDINGS_VERSION`   | Explicit version of Spring Cloud Bindings library to install.                                                                                                                                                                                                           |
| `$BP_SPRING_AOT_ENABLED`              | Whether to contribute `$BPL_SPRING_AOT_ENABLED` at runtime. Beware that the Spring Boot app needs to have been AOT instrumented (presence of `META-INF/native-image`) too. Defaults to false.                                                                           |
| `$BPL_SPRING_AOT_ENABLED`             | Whether to contribute `-Dspring.aot.enabled=true` to `JAVA_TOOL_OPTIONS` at runtime. Defaults to yes if the above conditions were met; false otherwise                                                                                                                  |                                                                                                           
| `$BP_JVM_CDS_ENABLED`                 | Whether to perform the CDS training run (that will generate the caching file `application.jsa`). Defaults to false.                                                                                                                                                     |
| `$CDS_TRAINING_JAVA_TOOL_OPTIONS`     | Allow the user to override the default `JAVA_TOOL_OPTIONS`, only for the CDS training run. Useful to configure your app not to reach external services during training run for example.                                                                                 |
| `$BPL_JVM_CDS_ENABLED`                | Whether to load the CDS caching file (`-XX:SharedArchiveFile=application.jsa`) that was generated during the CDS training run. Defaults to the value of `BP_JVM_CDS_ENABLED`                                                                                            |
## Bindings
The buildpack optionally accepts the following bindings:

### Type: `dependency-mapping`
| Key                   | Value   | Description                                                                                       |
| --------------------- | ------- | ------------------------------------------------------------------------------------------------- |
| `<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>` |

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0

