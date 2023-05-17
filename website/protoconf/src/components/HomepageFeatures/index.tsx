import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  img: string;
  description: JSX.Element;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Flexible',
    img: '/img/flexible.png',
    description: (
      <>
        Protoconf uses the Starlark language and supports multiple output formats, allowing dynamic and adaptable configuration generation for any application need.
      </>
    ),
  },
  {
    title: 'Safe',
    img: '/img/safe.png',
    description: (
      <>
        Protoconf enforces type safety with Protobufs and utilizes version-controlled configurations, ensuring valid and easily reversible changes.
      </>
    ),
  },
  {
    title: 'Fast',
    img: '/img/fast.png',
    description: (
      <>
        Protoconf automates many aspects of configuration management, enabling faster deployment processes and an accelerated development lifecycle.
      </>
    ),
  },
];

function Feature({ title, img, description }: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <img src={img} className={styles.featureSvg} />
      </div>
      <div className="text--center padding-horiz--md">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
