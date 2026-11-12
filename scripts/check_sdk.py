"""Compare official GeoIP2 SDK output with every distinct record in the artifact."""
import json
import sys

import geoip2.database
import maxminddb


def check(path):
    seen = set()
    provinces = set()
    cities = set()
    with maxminddb.open_database(path) as raw, geoip2.database.Reader(path) as sdk:
        assert sdk.metadata().database_type == 'GeoIP2-City'
        for network, record in raw:
            key = json.dumps(record, sort_keys=True)
            if key in seen:
                continue
            seen.add(key)
            ip = str(network.network_address)
            city = sdk.city(ip)
            assert city.country.iso_code == record['country']['iso_code']
            assert city.country.names == record['country']['names']
            expected = record.get('subdivisions', [])
            assert len(city.subdivisions) == len(expected)
            for actual, wanted in zip(city.subdivisions, expected):
                assert actual.iso_code == wanted['iso_code']
                assert actual.names == wanted['names']
                provinces.add(actual.iso_code)
            assert city.city.names == record.get('city', {}).get('names', {})
            assert city.city.geoname_id is None
            if city.city.names:
                assert city.subdivisions
                cities.add(city.city.names['zh-CN'])
            assert city.location.latitude is None
            assert city.registered_country.iso_code is None
        assert seen, 'empty database'
        assert provinces, 'no province records'
        assert cities, 'no city records'
    print(f'Official Python SDK: validated {len(seen)} distinct records, {len(provinces)} provinces, {len(cities)} cities')


if __name__ == '__main__':
    check(sys.argv[1])
