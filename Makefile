NAME=hodhod
BINDIR ?= .
SRC != find . -name '*.go' ! -name '*_test.go'

# Force using go's builtin dns resolver, instead of the system one, in order to
# produce a nice, clean, statically-linked executable!
FLAGS = -tags netgo

$(BINDIR)/$(NAME): $(SRC)
	go build -o $(BINDIR) $(FLAGS)

all: $(NAME)

release: $(NAME)
	strip $(NAME)

clean:
	rm -f $(BINDIR)/$(NAME)

.PHONY: all clean
