package v7pushaction_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v7action"
	. "code.cloudfoundry.org/cli/actor/v7pushaction"
	"code.cloudfoundry.org/cli/actor/v7pushaction/v7pushactionfakes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"code.cloudfoundry.org/cli/types"
	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func actualizedStreamsDrainedAndClosed(
	configStream <-chan PushPlan,
	eventStream <-chan Event,
	warningsStream <-chan Warnings,
	errorStream <-chan error,
) bool {
	var configStreamClosed, eventStreamClosed, warningsStreamClosed, errorStreamClosed bool
	for {
		select {
		case _, ok := <-configStream:
			if !ok {
				configStreamClosed = true
			}
		case _, ok := <-eventStream:
			if !ok {
				eventStreamClosed = true
			}
		case _, ok := <-warningsStream:
			if !ok {
				warningsStreamClosed = true
			}
		case _, ok := <-errorStream:
			if !ok {
				errorStreamClosed = true
			}
		}
		if configStreamClosed && eventStreamClosed && warningsStreamClosed && errorStreamClosed {
			break
		}
	}
	return true
}

// TODO: for refactor: We can use the following style of code to validate that
// each event is received in a specific order

// Expect(nextEvent()).Should(Equal(SettingUpApplication))
// Expect(nextEvent()).Should(Equal(CreatingApplication))
// Expect(nextEvent()).Should(Equal(...))
// Expect(nextEvent()).Should(Equal(...))
// Expect(nextEvent()).Should(Equal(...))
func getNextEvent(c <-chan PushPlan, e <-chan Event, w <-chan Warnings) func() Event {
	timeOut := time.Tick(500 * time.Millisecond)

	return func() Event {
		for {
			select {
			case <-c:
			case event, ok := <-e:
				if ok {
					log.WithField("event", event).Debug("getNextEvent")
					return event
				}
				return "getNextEvent closed"
			case <-w:
			case <-timeOut:
				return "getNextEvent timedout"
			}
		}
	}
}

var _ = Describe("Actualize", func() {
	var (
		actor           *Actor
		fakeV2Actor     *v7pushactionfakes.FakeV2Actor
		fakeV7Actor     *v7pushactionfakes.FakeV7Actor
		fakeSharedActor *v7pushactionfakes.FakeSharedActor

		plan            PushPlan
		fakeProgressBar *v7pushactionfakes.FakeProgressBar

		planStream     <-chan PushPlan
		eventStream    <-chan Event
		warningsStream <-chan Warnings
		errorStream    <-chan error
	)

	BeforeEach(func() {
		fakeV2Actor = new(v7pushactionfakes.FakeV2Actor)
		fakeV7Actor = new(v7pushactionfakes.FakeV7Actor)
		fakeSharedActor = new(v7pushactionfakes.FakeSharedActor)
		fakeSharedActor.ReadArchiveReturns(new(v7pushactionfakes.FakeReadCloser), 0, nil)
		actor = NewActor(fakeV2Actor, fakeV7Actor, fakeSharedActor)

		fakeProgressBar = new(v7pushactionfakes.FakeProgressBar)
		plan = PushPlan{
			Application: v7action.Application{
				Name: "some-app",
				GUID: "some-app-guid",
			},
			SpaceGUID: "some-space-guid",
		}

		fakeV2Actor.GetOrganizationDomainsReturns(
			[]v2action.Domain{
				{
					GUID: "some-domain-guid",
					Name: "some-domain",
				},
			},
			v2action.Warnings{"domain-warning"},
			nil,
		)
	})

	AfterEach(func() {
		Eventually(actualizedStreamsDrainedAndClosed(planStream, eventStream, warningsStream, errorStream)).Should(BeTrue())
	})

	JustBeforeEach(func() {
		planStream, eventStream, warningsStream, errorStream = actor.Actualize(plan, fakeProgressBar)
	})

	Describe("application", func() {
		BeforeEach(func() {
			plan.Application.GUID = "some-app-guid"
		})

		When("the apps needs an update", func() {
			BeforeEach(func() {
				plan.ApplicationNeedsUpdate = true
			})

			When("updating is successful", func() {
				BeforeEach(func() {
					fakeV7Actor.UpdateApplicationReturns(
						v7action.Application{
							Name:                "some-app",
							GUID:                "some-app-guid",
							LifecycleBuildpacks: []string{"some-buildpack-1"},
						},
						v7action.Warnings{"some-app-update-warnings"},
						nil)
				})

				It("puts the updated application in the stream", func() {
					Eventually(warningsStream).Should(Receive(ConsistOf("some-app-update-warnings")))

					Eventually(planStream).Should(Receive(MatchFields(IgnoreExtras,
						Fields{
							"Application": Equal(v7action.Application{
								Name:                "some-app",
								GUID:                "some-app-guid",
								LifecycleBuildpacks: []string{"some-buildpack-1"},
							}),
						})))
				})
			})

			When("updating errors", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("some-error")
					fakeV7Actor.UpdateApplicationReturns(
						v7action.Application{},
						v7action.Warnings{"some-app-update-warnings"},
						expectedErr)
				})

				It("returns the warnings and error", func() {
					Eventually(warningsStream).Should(Receive(ConsistOf("some-app-update-warnings")))
					Eventually(errorStream).Should(Receive(MatchError(expectedErr)))
				})
			})
		})

		When("the plan does not need an app update", func() {
			BeforeEach(func() {
				plan.ApplicationNeedsUpdate = false
			})

			It("does not update the application", func() {
				Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(SkippingApplicationCreation))
				Consistently(fakeV7Actor.UpdateApplicationCallCount).Should(Equal(0))
			})
		})
	})

	Describe("scaling the web process", func() {
		When("a scale override is passed", func() {
			When("the scale is successful", func() {
				var memory types.NullUint64

				BeforeEach(func() {
					plan.Application.GUID = "some-app-guid"

					plan.ScaleWebProcessNeedsUpdate = true
					memory = types.NullUint64{IsSet: true, Value: 2048}
					plan.ScaleWebProcess = v7action.Process{
						MemoryInMB: memory,
					}

					fakeV7Actor.ScaleProcessByApplicationReturns(v7action.Warnings{"scaling-warnings"}, nil)
					fakeV7Actor.UpdateApplicationReturns(
						v7action.Application{
							Name: "some-app",
							GUID: plan.Application.GUID,
						},
						v7action.Warnings{"some-app-update-warnings"},
						nil)
				})

				It("returns warnings and continues", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(ScaleWebProcess))
					Eventually(warningsStream).Should(Receive(ConsistOf("scaling-warnings")))
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(ScaleWebProcessComplete))

					Expect(fakeV7Actor.ScaleProcessByApplicationCallCount()).To(Equal(1))
					passedAppGUID, passedProcess := fakeV7Actor.ScaleProcessByApplicationArgsForCall(0)
					Expect(passedAppGUID).To(Equal("some-app-guid"))
					Expect(passedProcess).To(MatchFields(IgnoreExtras,
						Fields{
							"MemoryInMB": Equal(memory),
						}))
				})
			})

			When("the scale errors", func() {
				var expectedErr error

				BeforeEach(func() {
					plan.ScaleWebProcessNeedsUpdate = true

					expectedErr = errors.New("nopes")
					fakeV7Actor.ScaleProcessByApplicationReturns(v7action.Warnings{"scaling-warnings"}, expectedErr)
				})

				It("returns warnings and an error", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(ScaleWebProcess))
					Eventually(warningsStream).Should(Receive(ConsistOf("scaling-warnings")))
					Eventually(errorStream).Should(Receive(MatchError(expectedErr)))
					Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(ScaleWebProcessComplete))
				})
			})
		})

		When("a scale override is not provided", func() {
			It("should not scale the application", func() {
				Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(ScaleWebProcess))
				Consistently(fakeV7Actor.ScaleProcessByApplicationCallCount).Should(Equal(0))
			})
		})
	})

	Describe("setting process configuration", func() {
		When("process configuration is provided", func() {
			var startCommand types.FilteredString

			BeforeEach(func() {
				plan.UpdateWebProcessNeedsUpdate = true

				startCommand = types.FilteredString{IsSet: true, Value: "some-start-command"}
				plan.UpdateWebProcess = v7action.Process{
					Command: startCommand,
				}
			})

			When("the update is successful", func() {
				BeforeEach(func() {
					plan.Application.GUID = "some-app-guid"

					fakeV7Actor.UpdateApplicationReturns(
						v7action.Application{
							Name: "some-app",
							GUID: plan.Application.GUID,
						},
						v7action.Warnings{"some-app-update-warnings"},
						nil)

					fakeV7Actor.UpdateProcessByTypeAndApplicationReturns(v7action.Warnings{"health-check-warnings"}, nil)
				})

				It("sets the process config and returns warnings", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SetProcessConfiguration))
					Eventually(warningsStream).Should(Receive(ConsistOf("health-check-warnings")))
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SetProcessConfigurationComplete))

					Expect(fakeV7Actor.UpdateProcessByTypeAndApplicationCallCount()).To(Equal(1))
					passedProcessType, passedAppGUID, passedProcess := fakeV7Actor.UpdateProcessByTypeAndApplicationArgsForCall(0)
					Expect(passedProcessType).To(Equal(constant.ProcessTypeWeb))
					Expect(passedAppGUID).To(Equal("some-app-guid"))
					Expect(passedProcess).To(MatchFields(IgnoreExtras,
						Fields{
							"Command": Equal(startCommand),
						}))
				})
			})

			When("the update errors", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("nopes")
					fakeV7Actor.UpdateProcessByTypeAndApplicationReturns(v7action.Warnings{"health-check-warnings"}, expectedErr)
				})

				It("returns warnings and an error", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SetProcessConfiguration))
					Eventually(warningsStream).Should(Receive(ConsistOf("health-check-warnings")))
					Eventually(errorStream).Should(Receive(MatchError(expectedErr)))
					Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(SetProcessConfigurationComplete))
				})
			})
		})

		When("process configuration is not provided", func() {
			It("should not set the configuration", func() {
				Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(SetProcessConfiguration))
				Consistently(fakeV7Actor.UpdateProcessByTypeAndApplicationCallCount).Should(Equal(0))
			})
		})
	})

	Describe("default route creation", func() {
		When("creating a default route", func() {
			BeforeEach(func() {
				plan.SkipRouteCreation = false
			})

			When("route creation and mapping is successful", func() {
				BeforeEach(func() {
					fakeV2Actor.FindRouteBoundToSpaceWithSettingsReturns(
						v2action.Route{},
						v2action.Warnings{"route-warning"},
						actionerror.RouteNotFoundError{},
					)

					fakeV2Actor.CreateRouteReturns(
						v2action.Route{
							GUID: "some-route-guid",
							Host: "some-app",
							Domain: v2action.Domain{
								Name: "some-domain",
								GUID: "some-domain-guid",
							},
							SpaceGUID: "some-space-guid",
						},
						v2action.Warnings{"route-create-warning"},
						nil,
					)

					fakeV2Actor.MapRouteToApplicationReturns(
						v2action.Warnings{"map-warning"},
						nil,
					)
				})

				It("creates the route, maps it to the app, and returns any warnings", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingAndMappingRoutes))
					Eventually(warningsStream).Should(Receive(ConsistOf("domain-warning", "route-warning", "route-create-warning", "map-warning")))
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatedRoutes))
				})
			})

			When("route creation and mapping errors", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("some route error")
					fakeV2Actor.GetOrganizationDomainsReturns(
						[]v2action.Domain{
							{
								GUID: "some-domain-guid",
								Name: "some-domain",
							},
						},
						v2action.Warnings{"domain-warning"},
						expectedErr,
					)
				})

				It("returns errors and warnings", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingAndMappingRoutes))
					Eventually(warningsStream).Should(Receive(ConsistOf("domain-warning")))
					Eventually(errorStream).Should(Receive(MatchError(expectedErr)))
					Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(CreatedRoutes))
				})
			})
		})

		When("skipping default route creation", func() {
			BeforeEach(func() {
				plan.SkipRouteCreation = true
			})

			It("never attempts to create a route", func() {
				Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Or(Equal(CreatingAndMappingRoutes), Equal(CreatedRoutes)))
				Consistently(fakeV2Actor.GetApplicationRoutesCallCount).Should(BeZero())
				Consistently(fakeV2Actor.CreateRouteCallCount).Should(BeZero())
			})
		})
	})

	Describe("package upload", func() {
		When("docker image is provided", func() {
			BeforeEach(func() {
				plan.DockerImageCredentialsNeedsUpdate = true
				plan.DockerImageCredentials.Path = "some-docker-image"
				plan.DockerImageCredentials.Password = "some-docker-password"
				plan.DockerImageCredentials.Username = "some-docker-username"

				fakeV7Actor.CreateApplicationInSpaceReturns(
					v7action.Application{
						GUID:          "some-app-guid",
						Name:          plan.Application.Name,
						LifecycleType: constant.AppLifecycleTypeDocker,
					},
					v7action.Warnings{"some-app-warnings"},
					nil)
			})

			When("creating the package is successful", func() {
				BeforeEach(func() {
					fakeV7Actor.CreateDockerPackageByApplicationReturns(
						v7action.Package{GUID: "some-package-guid"},
						v7action.Warnings{"some-package-warnings"},
						nil)
				})

				It("sets the docker image", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SetDockerImage))
					Eventually(fakeV7Actor.CreateDockerPackageByApplicationCallCount).Should(Equal(1))
					Eventually(warningsStream).Should(Receive(ConsistOf("some-package-warnings")))
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SetDockerImageComplete))

					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StartingStaging))
					Eventually(fakeV7Actor.StageApplicationPackageCallCount).Should(Equal(1))

					appGUID, dockerCredentials := fakeV7Actor.CreateDockerPackageByApplicationArgsForCall(0)
					Expect(appGUID).To(Equal("some-app-guid"))
					Expect(dockerCredentials).To(MatchFields(IgnoreExtras,
						Fields{
							"Path":     Equal("some-docker-image"),
							"Username": Equal("some-docker-username"),
							"Password": Equal("some-docker-password"),
						}))

					Expect(fakeV7Actor.PollPackageArgsForCall(0)).To(MatchFields(IgnoreExtras,
						Fields{
							"GUID": Equal("some-package-guid"),
						}))
				})

				It("does not create/upload archive", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(Complete))
					Expect(fakeSharedActor.ZipDirectoryResourcesCallCount()).To(Equal(0))
					Expect(fakeV7Actor.CreateBitsPackageByApplicationCallCount()).To(Equal(0))
				})
			})

			When("creating the package errors", func() {
				var someErr error

				BeforeEach(func() {
					someErr = errors.New("I AM A BANANA")
					fakeV7Actor.CreateDockerPackageByApplicationReturns(v7action.Package{}, v7action.Warnings{"some-package-warnings"}, someErr)
				})

				It("returns errors and warnings", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SetDockerImage))
					Eventually(warningsStream).Should(Receive(ConsistOf("some-package-warnings")))
					Eventually(errorStream).Should(Receive(MatchError(someErr)))
					Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(SetDockerImageComplete))
				})
			})
		})

		When("uploading application bits", func() {
			BeforeEach(func() {
				plan = PushPlan{
					Application: v7action.Application{
						Name: "some-app",
						GUID: "some-app-guid",
					},
					BitsPath: "/some-bits-path",
					AllResources: []sharedaction.Resource{
						{Filename: "some-filename", Size: 6},
					},
					MatchedResources: []sharedaction.Resource{
						{Filename: "some-matched-filename", Size: 6},
					},
				}
			})

			When("the bits path is an archive", func() {
				BeforeEach(func() {
					plan.Archive = true
				})

				It("creates the archive", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingArchive))

					Eventually(fakeSharedActor.ZipArchiveResourcesCallCount).Should(Equal(1))
					bitsPath, resources := fakeSharedActor.ZipArchiveResourcesArgsForCall(0)
					Expect(bitsPath).To(Equal("/some-bits-path"))
					Expect(resources).To(ConsistOf(sharedaction.Resource{
						Filename: "some-filename",
						Size:     6,
					}))
				})
			})

			When("The bits path is a directory", func() {
				It("creates the archive", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingArchive))

					Eventually(fakeSharedActor.ZipDirectoryResourcesCallCount).Should(Equal(1))
					bitsPath, resources := fakeSharedActor.ZipDirectoryResourcesArgsForCall(0)
					Expect(bitsPath).To(Equal("/some-bits-path"))
					Expect(resources).To(ConsistOf(sharedaction.Resource{
						Filename: "some-filename",
						Size:     6,
					}))
				})
			})

			When("the archive creation is successful", func() {
				BeforeEach(func() {
					fakeSharedActor.ZipDirectoryResourcesReturns("/some/archive/path", nil)
					fakeV7Actor.UpdateApplicationReturns(
						v7action.Application{
							Name: "some-app",
							GUID: plan.Application.GUID,
						},
						v7action.Warnings{"some-app-update-warnings"},
						nil)
				})

				It("creates the package", func() {
					Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingPackage))

					Eventually(fakeV7Actor.CreateBitsPackageByApplicationCallCount).Should(Equal(1))
					Expect(fakeV7Actor.CreateBitsPackageByApplicationArgsForCall(0)).To(Equal("some-app-guid"))
				})

				When("the package creation is successful", func() {
					BeforeEach(func() {
						fakeV7Actor.CreateBitsPackageByApplicationReturns(v7action.Package{GUID: "some-guid"}, v7action.Warnings{"some-create-package-warning"}, nil)
					})

					It("reads the archive", func() {
						Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(ReadingArchive))
						Eventually(fakeSharedActor.ReadArchiveCallCount).Should(Equal(1))
						Expect(fakeSharedActor.ReadArchiveArgsForCall(0)).To(Equal("/some/archive/path"))
					})

					When("reading the archive is successful", func() {
						BeforeEach(func() {
							fakeReadCloser := new(v7pushactionfakes.FakeReadCloser)
							fakeSharedActor.ReadArchiveReturns(fakeReadCloser, 6, nil)
						})

						It("uploads the bits package", func() {
							Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadingApplicationWithArchive))
							Eventually(fakeV7Actor.UploadBitsPackageCallCount).Should(Equal(1))
							pkg, resource, _, size := fakeV7Actor.UploadBitsPackageArgsForCall(0)

							Expect(pkg).To(Equal(v7action.Package{GUID: "some-guid"}))
							Expect(resource).To(ConsistOf(sharedaction.Resource{
								Filename: "some-matched-filename",
								Size:     6,
							}))
							Expect(size).To(BeNumerically("==", 6))
						})

						When("the upload is successful", func() {
							BeforeEach(func() {
								fakeV7Actor.UploadBitsPackageReturns(v7action.Package{GUID: "some-guid"}, v7action.Warnings{"some-upload-package-warning"}, nil)
							})

							It("returns an upload complete event and warnings", func() {
								Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadingApplicationWithArchive))
								Eventually(warningsStream).Should(Receive(ConsistOf("some-upload-package-warning")))
								Eventually(eventStream).Should(Receive(Equal(UploadWithArchiveComplete)))
							})

							When("the upload errors", func() {
								When("the upload error is a retryable error", func() {
									var someErr error

									BeforeEach(func() {
										someErr = errors.New("I AM A BANANA")
										fakeV7Actor.UploadBitsPackageReturns(v7action.Package{}, v7action.Warnings{"upload-warnings-1", "upload-warnings-2"}, ccerror.PipeSeekError{Err: someErr})
									})

									It("should send a RetryUpload event and retry uploading", func() {
										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadingApplicationWithArchive))
										Eventually(warningsStream).Should(Receive(ConsistOf("upload-warnings-1", "upload-warnings-2")))
										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(RetryUpload))

										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadingApplicationWithArchive))
										Eventually(warningsStream).Should(Receive(ConsistOf("upload-warnings-1", "upload-warnings-2")))
										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(RetryUpload))

										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadingApplicationWithArchive))
										Eventually(warningsStream).Should(Receive(ConsistOf("upload-warnings-1", "upload-warnings-2")))
										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(RetryUpload))

										Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(EqualEither(RetryUpload, UploadWithArchiveComplete, Complete))
										Eventually(fakeV7Actor.UploadBitsPackageCallCount).Should(Equal(3))
										Expect(errorStream).To(Receive(MatchError(actionerror.UploadFailedError{Err: someErr})))
									})

								})

								When("the upload error is not a retryable error", func() {
									BeforeEach(func() {
										fakeV7Actor.UploadBitsPackageReturns(v7action.Package{}, v7action.Warnings{"upload-warnings-1", "upload-warnings-2"}, errors.New("dios mio"))
									})

									It("sends warnings and errors, then stops", func() {
										Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadingApplicationWithArchive))
										Eventually(warningsStream).Should(Receive(ConsistOf("upload-warnings-1", "upload-warnings-2")))
										Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(EqualEither(RetryUpload, UploadWithArchiveComplete, Complete))
										Eventually(errorStream).Should(Receive(MatchError("dios mio")))
									})
								})
							})
						})

						When("reading the archive fails", func() {
							BeforeEach(func() {
								fakeSharedActor.ReadArchiveReturns(nil, 0, errors.New("the bits"))
							})

							It("returns an error", func() {
								Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(ReadingArchive))
								Eventually(errorStream).Should(Receive(MatchError("the bits")))
							})
						})
					})

					When("the package creation errors", func() {
						BeforeEach(func() {
							fakeV7Actor.CreateBitsPackageByApplicationReturns(v7action.Package{}, v7action.Warnings{"package-creation-warning"}, errors.New("the bits"))
						})

						It("it returns errors and warnings", func() {
							Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingPackage))

							Eventually(warningsStream).Should(Receive(ConsistOf("package-creation-warning")))
							Eventually(errorStream).Should(Receive(MatchError("the bits")))
						})
					})
				})

				When("the archive creation errors", func() {
					BeforeEach(func() {
						fakeSharedActor.ZipDirectoryResourcesReturns("", errors.New("oh no"))
					})

					It("returns an error and exits", func() {
						Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(CreatingArchive))

						Eventually(errorStream).Should(Receive(MatchError("oh no")))
					})
				})
			})
		})
	})

	Describe("polling package", func() {
		When("the the polling is succesful", func() {
			BeforeEach(func() {
				fakeV7Actor.PollPackageReturns(v7action.Package{}, v7action.Warnings{"some-poll-package-warning"}, nil)
			})

			It("returns warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadWithArchiveComplete))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-poll-package-warning")))
			})
		})

		When("the the polling returns an error", func() {
			var someErr error

			BeforeEach(func() {
				someErr = errors.New("I AM A BANANA")
				fakeV7Actor.PollPackageReturns(v7action.Package{}, v7action.Warnings{"some-poll-package-warning"}, someErr)
			})

			It("returns errors and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(UploadWithArchiveComplete))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-poll-package-warning")))
				Eventually(errorStream).Should(Receive(MatchError(someErr)))
			})
		})
	})

	Describe("staging package", func() {
		BeforeEach(func() {
			fakeV7Actor.PollPackageReturns(v7action.Package{GUID: "some-pkg-guid"}, nil, nil)
		})

		It("stages the application using the package guid", func() {
			Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StartingStaging))
			Eventually(fakeV7Actor.StageApplicationPackageCallCount).Should(Equal(1))
			Expect(fakeV7Actor.StageApplicationPackageArgsForCall(0)).To(Equal("some-pkg-guid"))
		})

		When("staging is successful", func() {
			BeforeEach(func() {
				fakeV7Actor.StageApplicationPackageReturns(v7action.Build{GUID: "some-build-guid"}, v7action.Warnings{"some-staging-warning"}, nil)
			})

			It("returns a polling build event and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StartingStaging))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-staging-warning")))
				Eventually(eventStream).Should(Receive(Equal(PollingBuild)))
			})
		})

		When("staging errors", func() {
			BeforeEach(func() {
				fakeV7Actor.StageApplicationPackageReturns(v7action.Build{}, v7action.Warnings{"some-staging-warning"}, errors.New("ahhh, i failed"))
			})

			It("returns errors and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StartingStaging))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-staging-warning")))
				Eventually(errorStream).Should(Receive(MatchError("ahhh, i failed")))
			})
		})
	})

	Describe("no start", func() {
		When("The no start flag is provided", func() {
			BeforeEach(func() {
				plan.NoStart = true
			})

			When("The app is stopped", func() {
				BeforeEach(func() {
					plan.Application.State = constant.ApplicationStopped
				})

				It("Uploads a package and exits", func() {
					Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(StartingStaging))
					Expect(fakeV7Actor.StageApplicationPackageCallCount()).To(BeZero())
				})
			})

			When("The app is running", func() {
				BeforeEach(func() {
					fakeV7Actor.StopApplicationReturns(v7action.Warnings{"some-stopping-warning"}, nil)
					plan.Application.State = constant.ApplicationStarted
				})

				When("Stopping the app succeeds", func() {
					It("Uploads a package and exits", func() {
						Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StoppingApplication))
						Eventually(warningsStream).Should(Receive(ConsistOf("some-stopping-warning")))
						Eventually(eventStream).Should(Receive(Equal(StoppingApplicationComplete)))
						Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(StartingStaging))

						Expect(fakeV7Actor.StopApplicationCallCount()).To(Equal(1))
						actualGUID := fakeV7Actor.StopApplicationArgsForCall(0)
						Expect(actualGUID).To(Equal("some-app-guid"))
						Expect(fakeV7Actor.StageApplicationPackageCallCount()).To(BeZero())
					})
				})

				When("Stopping the app fails", func() {
					BeforeEach(func() {
						fakeV7Actor.StopApplicationReturns(v7action.Warnings{"some-stopping-warning"}, errors.New("bummer"))
					})

					It("returns errors and warnings", func() {
						Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StoppingApplication))
						Eventually(warningsStream).Should(Receive(ConsistOf("some-stopping-warning")))
						Consistently(getNextEvent(planStream, eventStream, warningsStream)).ShouldNot(Equal(StartingStaging))
						Eventually(errorStream).Should(Receive(MatchError("bummer")))
					})
				})
			})
		})

		When("The no start flag is not provided", func() {
			It("stages the application using the package guid", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(StartingStaging))
				Eventually(fakeV7Actor.StageApplicationPackageCallCount).Should(Equal(1))
			})
		})
	})

	Describe("polling build", func() {
		When("the the polling is successful", func() {
			BeforeEach(func() {
				fakeV7Actor.PollBuildReturns(v7action.Droplet{}, v7action.Warnings{"some-poll-build-warning"}, nil)
			})

			It("returns a staging complete event and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(PollingBuild))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-poll-build-warning")))
				Eventually(eventStream).Should(Receive(Equal(StagingComplete)))
			})
		})

		When("the the polling returns an error", func() {
			var someErr error

			BeforeEach(func() {
				someErr = errors.New("I AM A BANANA")
				fakeV7Actor.PollBuildReturns(v7action.Droplet{}, v7action.Warnings{"some-poll-build-warning"}, someErr)
			})

			It("returns errors and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(PollingBuild))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-poll-build-warning")))
				Eventually(errorStream).Should(Receive(MatchError(someErr)))
			})
		})
	})

	Describe("setting droplet", func() {
		When("setting the droplet is successful", func() {
			BeforeEach(func() {
				fakeV7Actor.SetApplicationDropletReturns(v7action.Warnings{"some-set-droplet-warning"}, nil)
			})

			It("returns a SetDropletComplete event and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SettingDroplet))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-set-droplet-warning")))
				Eventually(eventStream).Should(Receive(Equal(SetDropletComplete)))
			})
		})

		When("setting the droplet errors", func() {
			BeforeEach(func() {
				fakeV7Actor.SetApplicationDropletReturns(v7action.Warnings{"some-set-droplet-warning"}, errors.New("the climate is arid"))
			})

			It("returns an error and warnings", func() {
				Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(SettingDroplet))
				Eventually(warningsStream).Should(Receive(ConsistOf("some-set-droplet-warning")))
				Eventually(errorStream).Should(Receive(MatchError("the climate is arid")))
			})
		})
	})

	When("all operations are finished", func() {
		It("returns a complete event", func() {
			Eventually(getNextEvent(planStream, eventStream, warningsStream)).Should(Equal(Complete))
		})
	})
})
