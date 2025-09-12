FROM postgres:16-alpine

ENV POSTGRES_DB=scraperdb
ENV POSTGRES_USER=scraperuser
ENV POSTGRES_PASSWORD=supersecret
ENV PGDATA=/var/lib/postgresql/data/pgdata

COPY init.sql /docker-entrypoint-initdb.d/

EXPOSE 5432

HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=5 \
CMD pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB} || exit 1

VOLUME ["/var/lib/postgres/data"]
