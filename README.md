# `paketo-buildpacks/spring-boot`
The Paketo Spring Boot Buildpack is a Cloud Native Buildpack that contributes Spring Boot dependency information and slices an application into multiple layers.

## Behavior
This buildpack will participate if all of the following conditions are met

* `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Version` entry

The buildpack will do the following:

* Contributes dependency information extract from Maven naming conventions to the images BOM
* If `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` contains a `Spring-Boot-Layers-Index` entry
  * Contributes application slices as defined by the index
* If `<APPLICATION_ROOT>/META-INF/MANIFEST.MF` does not a `Spring-Boot-Layers-Index` entry
  * Contributes application slices as defined by convention
    * `BOOT-INF/lib/*-[^SNAPSHOT].jar`
    * `BOOT-INF/lib/*-[SNAPSHOT].jar`
    * `META-INF/resources/**`, `resources/**`, `static/**`, `public/**`
    * `BOOT-INF/classes/**`

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
