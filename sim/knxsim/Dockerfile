FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 3671/udp
EXPOSE 9090/tcp
CMD ["python", "-u", "knxsim.py"]
