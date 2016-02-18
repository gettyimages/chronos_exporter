package main

import "testing"

func Test_rename_metric(t *testing.T) {
	cases := []struct {
		source string
		expect string
	}{
		{
			source: "Foo",
			expect: "foo",
		}, {
			source: "foo_bar",
			expect: "foo_bar",
		}, {
			source: "foo.bar",
			expect: "foo_bar",
		}, {
			source: "foo-bar",
			expect: "foo_bar",
		}, {
			source: "foo$bar",
			expect: "foo_bar",
		}, {
			source: "foo(bar)",
			expect: "foo_bar",
		},
	}

	for _, c := range cases {
		name, _ := renameMetric(c.source)
		if name != c.expect {
			t.Errorf("expected metric named %s, got %s", c.expect, name)
		}
	}
}

func Test_rename_jobs_metric(t *testing.T) {
	cases := []struct {
		source string
		expect string
		labels map[string]string
	}{
		{
			source: "jobs.run.foo.bar",
			expect: "jobs_run_foo",
			labels: map[string]string{
				"job": "bar",
			},
		},
	}

	for _, c := range cases {
		metricName, labels := renameMetric(c.source)
		if metricName != c.expect {
			t.Errorf("expected rate named %s, but was %s", c.expect, metricName)
		}
		for labelName, labelValue := range labels {
			if expectValue := c.labels[labelName]; labelValue != expectValue {
				t.Errorf("expected %s value %s, but was %s", labelName, expectValue, labelValue)
			}
		}
	}
}

func Test_rename_rate(t *testing.T) {
	cases := []struct {
		name   string
		expect string
	}{
		{
			name:   "mean_rate",
			expect: "mean",
		}, {
			name:   "m1_rate",
			expect: "1m",
		}, {
			name:   "m5_rate",
			expect: "5m",
		}, {
			name:   "m15_rate",
			expect: "15m",
		}, {
			name:   "foo",
			expect: "foo",
		},
	}

	for _, c := range cases {
		name := renameRate(c.name)
		if name != c.expect {
			t.Errorf("expected rate named %s, got %s", c.expect, name)
		}
	}
}
