import * as React from 'react';
import {
  Button,
  Card,
  CardActions,
  CardBody,
  CardHead,
  CardHeader,
  Dropdown,
  DropdownItem,
  Gallery,
  GalleryItem,
  KebabToggle,
  PageSection,
  PageSectionVariants,
  Text,
  TextContent,
  TextInput,
  Toolbar,
  ToolbarGroup,
  ToolbarItem
} from '@patternfly/react-core';
import {ColumnsIcon, ListIcon, WarningTriangleIcon} from '@patternfly/react-icons';


interface Props {
}

interface State {
}

const Applications: React.FunctionComponent<Props> = ({}) => {
  const applications = [
    {
      name: 'OpenShift',
      description: 'Red Hat OpenShift is an open source container application platform based on the Kubernetes container orchestrator for enterprise app development.'
    },
    {
      name: 'Quay',
      description: 'A private container registry that helps you store, build, and deploy container images while identifying potential security vulnerabilities.'
    },
  ];

  const showCards = () => {
    console.log("show cards");
  };

  const showList = () => {
    console.log("show list");
  };

  return (
    <React.Fragment>
      <PageSection variant={PageSectionVariants.light}>
        <TextContent>
          <Text component="h1">Applications</Text>
          <Text component="p">This is an overview of all applications' status.</Text>
        </TextContent>
        <Toolbar id="data-toolbar-group-types">
          <ToolbarGroup>
            <ToolbarItem>
              <TextInput type="text" aria-label="search application"/>
            </ToolbarItem>
            <ToolbarItem>
              <Dropdown isPlain toggle={<KebabToggle id="toggle-id-6"/>} isOpen={false} dropdownItems={[
                <DropdownItem key="link">Link</DropdownItem>
              ]}></Dropdown>
            </ToolbarItem>
          </ToolbarGroup>
          <ToolbarGroup>
            <ToolbarItem>
              <Button variant="plain" aria-label="Show Cards" onClick={showCards}>
                <ColumnsIcon/>
              </Button>
            </ToolbarItem>
            <ToolbarItem>
              <Button variant="plain" aria-label="Show List" onClick={showList}>
                <ListIcon/>
              </Button>
            </ToolbarItem>
          </ToolbarGroup>
        </Toolbar>
      </PageSection>
      <PageSection>
        <Gallery gutter="md">
          {applications.map((app) => (
            <React.Fragment key={app.name}>
              <GalleryItem style={{margin: 8}}>
                <Card isHoverable={true} key={app.name}>
                  <CardHead>
                    <CardActions>
                      <WarningTriangleIcon/>
                    </CardActions>
                    <CardHeader>{app.name}</CardHeader>
                  </CardHead>
                  <CardBody>{app.description}</CardBody>
                </Card>
              </GalleryItem>
            </React.Fragment>
          ))}
        </Gallery>
      </PageSection>
    </React.Fragment>
  );
};

export {Applications};
