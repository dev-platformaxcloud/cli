package v2_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/flag"
	. "code.cloudfoundry.org/cli/command/v2"
	"code.cloudfoundry.org/cli/command/v2/v2fakes"
	"code.cloudfoundry.org/cli/types"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("UpdateBuildpackCommand", func() {
	var (
		cmd             UpdateBuildpackCommand
		fakeSharedActor *commandfakes.FakeSharedActor
		testUI          *ui.UI
		input           *Buffer
		fakeActor       *v2fakes.FakeUpdateBuildpackActor
		fakeConfig      *commandfakes.FakeConfig
		args            flag.BuildpackName
		buildpackGUID   string

		executeErr  error
		expectedErr error
	)

	BeforeEach(func() {
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		input = NewBuffer()
		testUI = ui.NewTestUI(input, NewBuffer(), NewBuffer())
		fakeActor = new(v2fakes.FakeUpdateBuildpackActor)
		fakeConfig = new(commandfakes.FakeConfig)
		args.Buildpack = "some-bp"
		buildpackGUID = "some guid"

		cmd = UpdateBuildpackCommand{
			RequiredArgs: args,
			UI:           testUI,
			SharedActor:  fakeSharedActor,
			Actor:        fakeActor,
			Config:       fakeConfig,
		}
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	When("an error is encountered checking if the environment is setup correctly", func() {
		BeforeEach(func() {
			expectedErr = actionerror.NotLoggedInError{BinaryName: "some name"}
			fakeSharedActor.CheckTargetReturns(expectedErr)
		})

		It("returns an error", func() {
			Expect(executeErr).To(MatchError(expectedErr))
			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			checkTargetedOrgArg, checkTargetedSpaceArg := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(checkTargetedOrgArg).To(BeFalse())
			Expect(checkTargetedSpaceArg).To(BeFalse())
		})
	})

	When("the environment is setup correctly", func() {
		When("getting the current user fails", func() {
			BeforeEach(func() {
				expectedErr = errors.New("some-error that happened")
				fakeConfig.CurrentUserReturns(configv3.User{}, expectedErr)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(expectedErr))
			})
		})

		When("getting the current user succeeds", func() {
			var userName string

			BeforeEach(func() {
				userName = "some-user"
				fakeConfig.CurrentUserReturns(configv3.User{Name: userName}, nil)
			})

			When("updating the buildpack fails", func() {
				BeforeEach(func() {
					expectedErr = errors.New("update-error")
					fakeActor.UpdateBuildpackByNameReturns(
						"",
						v2action.Warnings{"update-bp-warning1", "update-bp-warning2"},
						expectedErr,
					)
				})

				It("returns the error and prints any warnings", func() {
					Expect(testUI.Err).To(Say("update-bp-warning1"))
					Expect(testUI.Err).To(Say("update-bp-warning2"))
					Expect(executeErr).To(MatchError(expectedErr))
				})
			})

			When("updating the buildpack succeeds", func() {
				BeforeEach(func() {
					fakeActor.UpdateBuildpackByNameReturns(
						buildpackGUID,
						v2action.Warnings{"update-bp-warning1", "update-bp-warning2"},
						nil,
					)
				})

				When("no arguments are specified", func() {
					It("makes the actor call to update the buildpack", func() {
						Expect(fakeActor.UpdateBuildpackByNameCallCount()).To(Equal(1))
						name, order := fakeActor.UpdateBuildpackByNameArgsForCall(0)
						Expect(name).To(Equal(args.Buildpack))
						Expect(order.IsSet).To(BeFalse())

						Expect(testUI.Err).To(Say("update-bp-warning1"))
						Expect(testUI.Err).To(Say("update-bp-warning2"))
						Expect(testUI.Out).To(Say("Updating buildpack some-bp as some-user..."))
						Expect(testUI.Out).To(Say("OK"))
					})
				})

				When("a path is specified", func() {
					BeforeEach(func() {
						cmd.Path = flag.PathWithExistenceCheckOrURL("some path")
					})

					It("makes the actor call to update the buildpack", func() {
						Expect(fakeActor.UpdateBuildpackByNameCallCount()).To(Equal(1))
						name, order := fakeActor.UpdateBuildpackByNameArgsForCall(0)
						Expect(name).To(Equal(args.Buildpack))
						Expect(order.IsSet).To(BeFalse())

						Expect(testUI.Err).To(Say("update-bp-warning1"))
						Expect(testUI.Err).To(Say("update-bp-warning2"))
						Expect(testUI.Out).To(Say("Updating buildpack some-bp as some-user..."))
						Expect(testUI.Out).To(Say("OK"))
					})

					When("preparing the bits fails", func() {
						BeforeEach(func() {
							expectedErr = errors.New("prepare error")
							fakeActor.PrepareBuildpackBitsReturns("", expectedErr)
						})

						It("returns an error", func() {
							Expect(executeErr).To(MatchError(expectedErr))
						})
					})

					When("preparing the bits succeeds", func() {
						buildpackBitsPath := "some path on the file system"
						BeforeEach(func() {
							fakeActor.PrepareBuildpackBitsReturns(buildpackBitsPath, nil)
						})

						It("uploads the new buildpack bits", func() {
							Expect(testUI.Out).To(Say("Uploading buildpack some-bp as some-user..."))
							Expect(fakeActor.UploadBuildpackCallCount()).To(Equal(1))
							buildpackGUIDUsed, pathUsed, _ := fakeActor.UploadBuildpackArgsForCall(0)
							Expect(buildpackGUIDUsed).To(Equal(buildpackGUID))
							Expect(pathUsed).To(Equal(buildpackBitsPath))
						})

						When("uploading the buildpack fails", func() {
							BeforeEach(func() {
								expectedErr = errors.New("upload error")
								fakeActor.UploadBuildpackReturns(v2action.Warnings{"upload-warning1", "upload-warning2"}, expectedErr)
							})

							It("returns warnings and an error", func() {
								Expect(testUI.Err).To(Say("upload-warning1"))
								Expect(testUI.Err).To(Say("upload-warning2"))
								Expect(executeErr).To(MatchError(expectedErr))
							})
						})

						When("uploading the buildpack succeeds", func() {
							BeforeEach(func() {
								fakeActor.UploadBuildpackReturns(v2action.Warnings{"upload-warning1", "upload-warning2"}, nil)
							})
							It("displays success test and any warnings", func() {
								Expect(testUI.Err).To(Say("upload-warning1"))
								Expect(testUI.Err).To(Say("upload-warning2"))
								Expect(testUI.Out).To(Say("Done uploading"))
								Expect(testUI.Out).To(Say("OK"))
							})
						})
					})
				})

				When("an order is specified", func() {
					BeforeEach(func() {
						cmd.Order = types.NullInt{Value: 3, IsSet: true}
					})

					It("makes the actor call to update the buildpack", func() {
						Expect(fakeActor.UpdateBuildpackByNameCallCount()).To(Equal(1))
						name, order := fakeActor.UpdateBuildpackByNameArgsForCall(0)
						Expect(name).To(Equal(args.Buildpack))
						Expect(order.IsSet).To(BeTrue())
						Expect(order.Value).To(Equal(3))

						Expect(testUI.Err).To(Say("update-bp-warning1"))
						Expect(testUI.Err).To(Say("update-bp-warning2"))
						Expect(testUI.Out).To(Say("Updating buildpack some-bp as some-user..."))
						Expect(testUI.Out).To(Say("OK"))
					})
				})
			})
		})
	})
})