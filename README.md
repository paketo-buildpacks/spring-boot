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
* If `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Layers-Index` entry
  * Contributes application slices as defined by the layer's index

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
