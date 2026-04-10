FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o eth-sweeper .

FROM python:3.11.9-slim

WORKDIR /app

# Install python dependencies for XGBoost model
RUN pip install --no-cache-dir pandas numpy xgboost scikit-learn

# Copy executable and ML resources
COPY --from=builder /app/eth-sweeper .
COPY --from=builder /app/ml ./ml

EXPOSE 8080
CMD ["./eth-sweeper"]
