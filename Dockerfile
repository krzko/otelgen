FROM golang:1.18.4-alpine@sha256:d84b1ff3eeb9404e0a7dda7fdc6914cbe657102420529beec62ccb3ef3d143eb

RUN apk add --no-cache bash \
	tini

# install cosign
COPY --from=gcr.io/projectsigstore/cosign:v1.7.2@sha256:ad2985a87622d5934a4bc06a61faadff772e377937e42519af4f506e1b019d1e /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
