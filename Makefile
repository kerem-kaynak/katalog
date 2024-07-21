PROJECT_ID=katalog-427312
IMAGE_NAME=katalog-backend
REGION=europe-west4
SERVICE_NAME=katalog-backend

GCR_IMAGE=gcr.io/$(PROJECT_ID)/$(IMAGE_NAME)

# Build the Docker image
build:
	docker build -t $(GCR_IMAGE) -f Dockerfile.prod .

# Push the Docker image to Google Container Registry
push:
	docker push $(GCR_IMAGE)

# Deploy the Docker image to Cloud Run
deploy:
	gcloud run deploy $(SERVICE_NAME) \
		--image $(GCR_IMAGE) \
		--platform managed \
		--region $(REGION) \

# Combined command to build, push, and deploy
all: build push deploy