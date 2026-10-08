# Quant Lab (stdlib-only Python): option analytics and forecasts published as metrics.
FROM python:3.13-alpine
COPY quantlab.py /app/quantlab.py
USER 65534
EXPOSE 9108
ENTRYPOINT ["python3", "/app/quantlab.py"]
CMD ["--prometheus=http://prometheus:9090", "--listen=:9108"]
