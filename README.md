<img src="logo.png" alt="Salta" width="500px"/>

A small footprint reverse-geocoder for when you don't need precision.

## Goal

Salta is a simple reverse geocoder for when you don't need precision, i.e. the most precise location you need is at the city or suburb level.

The aim is to have:

- low memory usage
- fast start-up time
- no database
- fast updates

Salta relies on the [Who's On First](https://whosonfirst.org/) database.

## Performance

In order to optimize memory usage, on the first launch Salta simplifies polygons
and stores a cached version of every processed geojson.
On subsequent launches cached files are used when the source hasn't changed.

| Country       | Load time (with cache) | Initial load time (no cache) | Memory usage | Total disk usage (repo/cache) |
| ------------- | ---------------------- | ---------------------------- | ------------ | ----------------------------- |
| New Zealand   | 1s                     | 4mn                          | 23MB         | 791MB (738MB/53MB)            |
| France        | 3s                     | 3mn                          | 152MB        | 4GB (2.9G/914MB)              |
| United States | 6s                     | 15mn                         | 259MB        | 8.6GB (6.8GB/1.8GB)           |

*Loading all place types. Memory usage after GC. SSD drive, AMD Ryzen 3700X.*

When using the **cache only mode** the initial load time can be significantly faster, when loading the **whole world** is takes:
- A few seconds to load countries and localities
- Around five minutes to load all place types

In terms of **memory usage for loading the whole world**, depending on the place types loaded:

| Place type                       | Memory usage |
| -------------------------------- | ------------ |
| Countries                        | 73MB         |
| Countries + Localities           | 512MB        |
| Countries + Regions + Localities | 818MB        |
| All                              | 1.3GB        |

The **mean response time for HTTP requests** is around 200-300μs.

## Usage

### Config

Sample config file:

```yaml
# In cache only mode Salta only loads the available cache files. This makes
# start-up significantly faster.
cache_only: true # default: false
countries: # default: all countries
  - nz
  - fr
cache:
  folder: /path/to/salta/cache/folder # default: cache
repos:
  folder: /path/to/salta/repos/folder # default: repos
enabled_place_types: # default: all
  - locality
  - neighbourhood
  - borough
  - microhood
  - county
  - macrocounty
  - localadmin
  - region
  - macroregion
  - country
  - campus
  - marketarea
```

Supported formats: JSON, YAML.

### Run

#### With Docker

```sh
docker run -v $PWD/config.yaml:/config.yaml -t ghcr.io/pixwire/salta:latest
```

## Used by

Salta was created for use in [Pixwire](https://pixwire.net).

<a href="https://pixwire.net"><img src="https://raw.githubusercontent.com/pixwire/.github/main/pixwire-gopher.jpg" width="60%" /></a>
