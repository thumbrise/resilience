import Blueprint from './components/blueprint/Blueprint.tsx';
import { Button, Layout, Space } from 'antd';
import 'antd/dist/reset.css';

export default function App() {
  const handleSave = (data: any) => {
    console.log('handle save', data);
  };
  const handleLoad = (data: any) => {
    console.log('handle load', data);
  };
  return (
    <Layout style={{ height: '100vh' }}>
      <Layout.Sider width="25%" style={{ background: '#1e1e2e', padding: '16px' }}>
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Button type="primary" block onClick={handleSave}>
            Save
          </Button>
          <Button block onClick={handleLoad}>
            Load
          </Button>
        </Space>
      </Layout.Sider>
      <Layout.Content>
        <Blueprint />
      </Layout.Content>
    </Layout>
  );
}
