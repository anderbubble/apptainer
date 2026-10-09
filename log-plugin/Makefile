V := @

PLUGIN_NAME := apptainer.org/log-plugin
PLUGIN_SHORTNAME := log-plugin
PLUGIN_SIF := $(PLUGIN_SHORTNAME).sif

APPTAINER_VER := $(shell apptainer version 2>/dev/null)
ifneq ($(APPTAINER_VER), )
APPTAINER_BUILDDIR := $(shell apptainer buildcfg | grep -oP '(?<=^BUILDDIR=).*')
APPTAINER_CONFDIR := $(shell apptainer buildcfg | grep -oP '(?<=^APPTAINER_CONFDIR=).*')
APPTAINER_SOURCEDIR := $(shell dirname $(APPTAINER_BUILDDIR))
PLUGIN_CONFDIR := $(APPTAINER_CONFDIR)/plugins/$(PLUGIN_NAME)
endif


all: $(PLUGIN_SIF)

.PHONY: has_apptainer
has_apptainer:
ifeq ($(APPTAINER_VER), )
	@echo 'Please ensure apptainer is in your PATH'
	exit 1
else
	@echo 'Using Apptainer source directory: $(APPTAINER_SOURCEDIR)'
endif

$(PLUGIN_SIF): has_apptainer *.go
	@echo " COMPILE apptainer plugin compile"
	$(V)cd $(APPTAINER_SOURCEDIR) && apptainer plugin compile -o log-plugin/$(PLUGIN_SIF) log-plugin/

.PHONY: clean
clean:
	@echo " CLEAN"
	$(V)rm -f $(PLUGIN_SIF)

install: $(PLUGIN_SIF)
	@echo " INSTALL apptainer plugin install"
	$(V)apptainer plugin install $(PLUGIN_SIF)
	$(V)mkdir -p $(PLUGIN_CONFDIR)
	$(V)cp config.yml *.tpl $(PLUGIN_CONFDIR)
	$(V)apptainer plugin list

uninstall: has_apptainer
	@echo " UNINSTALL apptainer plugin uninstall"
	$(V)apptainer plugin disable $(PLUGIN_NAME)
	$(V)rm -f $(PLUGIN_CONFDIR)/config.yml $(PLUGIN_CONFDIR)/*.tpl
	$(V)apptainer plugin uninstall $(PLUGIN_NAME)

test: has_apptainer
	@echo " TEST go test"
	$(V)cd $(APPTAINER_SOURCEDIR) && scripts/go-test -count 1 -race -cover ./log-plugin/...

check: has_apptainer
	@echo " CHECK golangci-lint"
	$(V)cd $(APPTAINER_SOURCEDIR) && golangci-lint run ./log-plugin/...
