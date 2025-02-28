package benchmarks_test

import (
	"context"
	"fmt"
	"time"

	"github.com/observatorium/loki-benchmarks/internal/loadclient"
	"github.com/observatorium/loki-benchmarks/internal/metrics"
	"github.com/observatorium/loki-benchmarks/internal/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Scenario: High Volume Writes", func() {

	scenarioCfgs := benchCfg.Scenarios.HighVolumeWrites

	samplingCfg := scenarioCfgs.Samples.SamplingConfiguration()
	samplingRange := model.Duration(scenarioCfgs.Samples.Interval)

	BeforeEach(func() {
		if !scenarioCfgs.Enabled {
			Skip("High Volumes Writes Benchmark not enabled!")
		}
	})

	for _, scenarioCfg := range scenarioCfgs.Configurations {
		scenarioCfg := scenarioCfg
		generatorDpl := loadclient.CreateGenerator(scenarioCfg.Writers, benchCfg.Generator)

		Describe(fmt.Sprintf("Configuration: %s", scenarioCfg.Description), func() {
			BeforeEach(func() {
				err := k8sClient.Create(context.TODO(), generatorDpl, &client.CreateOptions{})
				Expect(err).Should(Succeed(), "Failed to deploy logger")

				err = utils.WaitForReadyDeployment(k8sClient, generatorDpl, defaultRetry, defaultTimeout)
				Expect(err).Should(Succeed(), "Failed to wait for ready logger deployment")

				// Sleeping for the first interval so that the data is accurate for the new workload.
				time.Sleep(scenarioCfgs.Samples.Interval)

				DeferCleanup(func() {
					err := k8sClient.Delete(context.TODO(), generatorDpl, &client.DeleteOptions{})
					Expect(err).Should(Succeed(), "Failed to delete logger deployment")
				})
			})

			It("should measure metrics", func() {
				e := gmeasure.NewExperiment(scenarioCfg.Description)
				AddReportEntry(e.Name, e)

				e.Sample(func(idx int) {
					// Distributors
					job := benchCfg.Metrics.Jobs.Distributor
					annotation := metrics.DistributorAnnotation

					// These are confirmation metrics to ensure that the workload matches expectations
					err := metricsClient.Measure(e, metrics.LoadNetworkTotal(loadclient.DeploymentName, samplingRange))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.LoadNetworkGiPDTotal(loadclient.DeploymentName, samplingRange))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.DistributorGiPDReceivedTotal(job, samplingRange))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))

					err = metricsClient.Measure(e, metrics.RequestWritesQPS(job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.RequestDurationOkPushAvg(job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.RequestDurationOkPushPercentile(99, job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.RequestDurationOkPushPercentile(50, job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))

					// Ingesters
					job = benchCfg.Metrics.Jobs.Ingester
					annotation = metrics.IngesterAnnotation

					err = metricsClient.Measure(e, metrics.ContainerCPU(job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))

					if benchCfg.Metrics.EnableCadvisorMetrics {
						err = metricsClient.Measure(e, metrics.ContainerMemoryWorkingSetBytes(job, samplingRange, annotation))
						Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					}

					err = metricsClient.Measure(e, metrics.PersistentVolumeUsedBytes(job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))

					err = metricsClient.Measure(e, metrics.RequestWritesGrpcQPS(job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.RequestDurationOkGrpcPushAvg(job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.RequestDurationOkGrpcPushPercentile(99, job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
					err = metricsClient.Measure(e, metrics.RequestDurationOkGrpcPushPercentile(50, job, samplingRange, annotation))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))

					err = metricsClient.Measure(e, metrics.RequestBoltDBShipperWritesQPS(job, samplingRange))
					Expect(err).Should(Succeed(), fmt.Sprintf("Failed - %v", err))
				}, samplingCfg)
			})
		})
	}
})
