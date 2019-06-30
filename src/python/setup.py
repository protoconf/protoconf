import setuptools

with open("README.md", "r") as fh:
    long_description = fh.read()

setuptools.setup(
    name="protoconf",
    version="0.0.1",
    author="Lior Tubi",
    author_email="lior2b@gmail.com",
    description="Codify configuration, instant delivery",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://gitlab.com/protoconf/protoconf/",
    packages=['protoconf', 'protoconf.agent.api.proto.v1'],
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
)
