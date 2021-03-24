# Call with `build_publish_docker.sh {version}`
# where {version} is in the 1.2.3 format

tokens=($(echo $1 | tr '.' '\n'))
major=${tokens[0]}
minor=${tokens[1]}
patch=${tokens[2]}

echo "Building plain image..."
docker build . -f Dockerfile_plain \
    -t frapasa/rept:latest \
    -t frapasa/rept:$major \
    -t frapasa/rept:$major.$minor \
    -t frapasa/rept:$major.$minor.$patch > /dev/null

echo "Pushing plain image..."
# docker image push --all-tags frapasa/rept

echo "Building debian image..."
docker build . -f Dockerfile_debian \
    -t frapasa/rept:latest-buster \
    -t frapasa/rept:$major-buster \
    -t frapasa/rept:$major.$minor-buster \
    -t frapasa/rept:$major.$minor.$patch-buster > /dev/null

echo "Pushing debian image..."
# docker image push --all-tags frapasa/rept

echo "Success!"
